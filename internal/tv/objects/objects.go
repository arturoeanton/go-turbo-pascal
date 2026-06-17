// Package objects implements the Turbo Vision Objects unit. The unit
// provides the foundation classes of the TV object model: TObject,
// TStream, TDosStream, TBufStream, TCollection, TSortedCollection,
// TStringCollection, TResourceCollection. BPGo reuses the sem.Type
// metadata to model VMTs and the ir.VM to dispatch methods.
package objects

import (
	"sync"
)

// TObject is the root class. It mirrors the TP7 object model: a VMT
// pointer at offset 0, instance fields after, and virtual Init/Done
// methods. The Go representation uses a Map of fields and a slice of
// method names.
type TObject struct {
	VMT      []string
	Fields   map[string]interface{}
	mu       sync.Mutex
	Instance bool
}

// Init constructs a new TObject.
func (o *TObject) Init() *TObject {
	o.VMT = []string{"Init", "Done", "Free"}
	o.Fields = map[string]interface{}{}
	o.Instance = true
	return o
}

// Done is the default destructor.
func (o *TObject) Done() {}

// Free releases the object.
func (o *TObject) Free() {
	o.Done()
	o.Instance = false
}

// Error is the default error handler.
func (o *TObject) Error() {}

// TStream is the abstract stream class.
type TStream struct {
	TObject
	Pos    int64
	Size   int64
	Status int
	Error  int
}

// Init constructs a TStream.
func (s *TStream) Init() *TStream {
	s.TObject.Init()
	return s
}

// Get reads bytes from the stream.
func (s *TStream) Get(p []byte) (int, error) { return 0, nil }

// Put writes bytes to the stream.
func (s *TStream) Put(p []byte) (int, error) { return len(p), nil }

// TBufStream is a buffered stream.
type TBufStream struct {
	TStream
	Buffer []byte
	Pos    int
}

// Init creates a new TBufStream.
func (b *TBufStream) Init() *TBufStream {
	b.TStream.Init()
	b.Buffer = []byte{}
	return b
}

// Flush writes pending data.
func (b *TBufStream) Flush() error { return nil }

// TDosStream is a DOS file stream.
type TDosStream struct {
	TStream
	Filename string
	Mode     int
}

// Init creates a new TDosStream.
func (d *TDosStream) Init(name string, mode int) *TDosStream {
	d.TStream.Init()
	d.Filename = name
	d.Mode = mode
	return d
}

// TCollection is a list of objects.
type TCollection struct {
	TObject
	Items []interface{}
	Limit int
	Delta int
}

// Init creates a new collection.
func (c *TCollection) Init(limit, delta int) *TCollection {
	c.TObject.Init()
	c.Limit = limit
	c.Delta = delta
	return c
}

// Count returns the number of items.
func (c *TCollection) Count() int { return len(c.Items) }

// GetItem returns the i-th item.
func (c *TCollection) GetItem(i int) interface{} {
	if i < 0 || i >= len(c.Items) {
		return nil
	}
	return c.Items[i]
}

// PutItem sets the i-th item.
func (c *TCollection) PutItem(i int, item interface{}) {
	if i < 0 {
		return
	}
	for i >= len(c.Items) {
		c.Items = append(c.Items, nil)
	}
	c.Items[i] = item
}

// At returns the i-th item.
func (c *TCollection) At(i int) interface{} { return c.GetItem(i) }

// Insert inserts an item at position i.
func (c *TCollection) Insert(item interface{}) {
	c.Items = append(c.Items, item)
}

// Delete removes the i-th item.
func (c *TCollection) Delete(i int) {
	if i < 0 || i >= len(c.Items) {
		return
	}
	c.Items = append(c.Items[:i], c.Items[i+1:]...)
}

// FreeAll deletes all items and the collection.
func (c *TCollection) FreeAll() {
	c.Items = nil
	c.Done()
}

// TSortedCollection sorts by a user-supplied comparison function.
type TSortedCollection struct {
	TCollection
	Compare func(a, b interface{}) int
}

// Init creates a new sorted collection.
func (s *TSortedCollection) Init(limit, delta int) *TSortedCollection {
	s.TCollection.Init(limit, delta)
	return s
}

// CompareKey returns a sortable key for an item.
func (s *TSortedCollection) CompareKey(k1, k2 interface{}) int {
	if s.Compare != nil {
		return s.Compare(k1, k2)
	}
	return 0
}

// Insert inserts an item keeping the collection sorted.
func (s *TSortedCollection) Insert(item interface{}) {
	pos := 0
	for pos < len(s.Items) {
		if s.Compare(item, s.Items[pos]) < 0 {
			break
		}
		pos++
	}
	s.Items = append(s.Items, nil)
	copy(s.Items[pos+1:], s.Items[pos:])
	s.Items[pos] = item
}

// TStringCollection stores strings.
type TStringCollection struct {
	TSortedCollection
}

// Init creates a new string collection.
func (s *TStringCollection) Init(limit, delta int) *TStringCollection {
	s.TSortedCollection.Init(limit, delta)
	s.Compare = func(a, b interface{}) int {
		as, bs := a.(string), b.(string)
		if as < bs {
			return -1
		}
		if as > bs {
			return 1
		}
		return 0
	}
	return s
}

// TResourceCollection stores resources.
type TResourceCollection struct {
	TCollection
}

// Init creates a new resource collection.
func (r *TResourceCollection) Init(limit, delta int) *TResourceCollection {
	r.TCollection.Init(limit, delta)
	return r
}

// TStrListMaker builds string lists.
type TStrListMaker struct {
	TStringCollection
}

// Init creates a new TStrListMaker.
func (m *TStrListMaker) Init(limit, delta int) *TStrListMaker {
	m.TStringCollection.Init(limit, delta)
	return m
}

// Put writes a string.
func (m *TStrListMaker) Put(s string) {
	m.Insert(s)
}

// Get retrieves a string.
func (m *TStrListMaker) Get(i int) string {
	v := m.GetItem(i)
	if v == nil {
		return ""
	}
	return v.(string)
}
