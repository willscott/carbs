package carbs

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipld/go-car"
	"github.com/multiformats/go-multihash"
)

func mkCar() (string, error) {
	f, err := ioutil.TempFile(os.TempDir(), "car")
	if err != nil {
		return "", err
	}
	defer f.Close()

	ds := mockNodeGetter{
		Nodes: make(map[cid.Cid]format.Node),
	}
	type linker struct {
		Name  string
		Links []*format.Link
	}
	cbornode.RegisterCborType(linker{})

	children := make([]format.Node, 0, 10)
	childLinks := make([]*format.Link, 0, 10)
	for i := 0; i < 10; i++ {
		child, _ := cbornode.WrapObject([]byte{byte(i)}, multihash.SHA2_256, -1)
		children = append(children, child)
		childLinks = append(childLinks, &format.Link{Name: fmt.Sprintf("child%d", i), Cid: child.Cid()})
	}
	b, err := cbornode.WrapObject(linker{Name: "root", Links: childLinks}, multihash.SHA2_256, -1)
	if err != nil {
		return "", err
	}
	ds.Nodes[b.Cid()] = b

	if err := car.WriteCar(context.Background(), &ds, []cid.Cid{b.Cid()}, f); err != nil {
		return "", err
	}

	return f.Name(), nil
}

func TestIndexRT(t *testing.T) {
	carFile, err := mkCar()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(carFile)

	cf, err := Load(carFile, false)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(carFile + ".idx")

	r, err := cf.Roots()
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 1 {
		t.Fatalf("unexpected number of roots: %d", len(r))
	}
	if _, err := cf.Get(r[0]); err != nil {
		t.Fatal(err)
	}

	idx, err := Restore(carFile)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Get(r[0]) != 0 {
		t.Fatalf("bad index: %d", idx.Get(r[0]))
	}
}

type mockNodeGetter struct {
	Nodes map[cid.Cid]format.Node
}

func (m *mockNodeGetter) Get(_ context.Context, c cid.Cid) (format.Node, error) {
	n, ok := m.Nodes[c]
	if !ok {
		return nil, fmt.Errorf("unknown node")
	}
	return n, nil
}

func (m *mockNodeGetter) GetMany(_ context.Context, cs []cid.Cid) <-chan *format.NodeOption {
	ch := make(chan *format.NodeOption, 5)
	go func() {
		for _, c := range cs {
			n, e := m.Get(nil, c)
			ch <- &format.NodeOption{n, e}
		}
	}()
	return ch
}