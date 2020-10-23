package carbs

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/ipfs/go-cid"
)

// IndexCodec is used as a multicodec identifier for carbs index files
type IndexCodec int

// IndexCodec table is a first var-int in carbs indexes
const (
	IndexHashed IndexCodec = iota + 0x300000
	IndexSorted
	IndexSingleSorted
)

// IndexCls is a constructor for an index type
type IndexCls func() Index

// IndexAtlas holds known index formats
var IndexAtlas = map[IndexCodec]IndexCls{
	IndexHashed:       mkHashed,
	IndexSorted:       mkSorted,
	IndexSingleSorted: mkSingleSorted,
}

// Record is a pre-processed record of a car item and location.
type Record struct {
	cid.Cid
	idx uint64
}

// Index provides an interface for figuring out where in the car a given cid begins
type Index interface {
	Codec() IndexCodec
	Marshal(w io.Writer) error
	Unmarshal(r io.Reader) error
	Get(cid.Cid) uint64
	Load([]Record) error
}

// Save writes a generated index for a car at `path`
func Save(i Index, path string) error {
	stream, err := os.OpenFile(path+".idx", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return err
	}
	defer stream.Close()

	buf := make([]byte, binary.MaxVarintLen64)
	b := binary.PutUvarint(buf, uint64(i.Codec()))
	if _, err := stream.Write(buf[:b]); err != nil {
		return err
	}
	return i.Marshal(stream)
}

// Restore loads an index from an on-disk representation.
func Restore(path string) (Index, error) {
	stream, err := os.Open(path + ".idx")
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	codec, err := binary.ReadUvarint(&unatreader{stream, 0})
	if err != nil {
		return nil, err
	}
	idx, ok := IndexAtlas[IndexCodec(codec)]
	if !ok {
		return nil, fmt.Errorf("Unknown codec: %d", codec)
	}
	idxInst := idx()
	if err := idxInst.Unmarshal(stream); err != nil {
		return nil, err
	}

	return idxInst, nil
}
