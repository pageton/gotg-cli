package invoke

import (
	"encoding/binary"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

func TestDecodeResponse_KnownConstructor(t *testing.T) {
	// Encode a known type: photos.photos with empty fields.
	photos := &tg.PhotosPhotos{}
	var buf bin.Buffer
	if err := photos.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	result, err := decodeResponse(buf.Buf)
	if result == nil {
		t.Fatalf("decodeResponse returned nil for known constructor: %v", err)
	}

	_, ok := result.(*tg.PhotosPhotos)
	if !ok {
		t.Errorf("expected *tg.PhotosPhotos, got %T", result)
	}
}

func TestDecodeResponse_BoolTrue(t *testing.T) {
	// Encode a simple type: boolTrue.
	b := &tg.BoolTrue{}
	var buf bin.Buffer
	if err := b.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	result, err := decodeResponse(buf.Buf)
	if result == nil {
		t.Fatalf("decodeResponse returned nil for boolTrue: %v", err)
	}
	_, ok := result.(*tg.BoolTrue)
	if !ok {
		t.Errorf("expected *tg.BoolTrue, got %T", result)
	}
}

func TestDecodeResponse_EmptyData(t *testing.T) {
	result, _ := decodeResponse(nil)
	if result != nil {
		t.Error("expected nil for nil data")
	}

	result, _ = decodeResponse([]byte{})
	if result != nil {
		t.Error("expected nil for empty data")
	}

	result, _ = decodeResponse([]byte{0x01, 0x02})
	if result != nil {
		t.Error("expected nil for < 4 bytes")
	}
}

func TestDecodeResponse_UnknownConstructor(t *testing.T) {
	// Use a fake constructor ID.
	data := make([]byte, 8)
	binary.LittleEndian.PutUint32(data[0:4], 0xDEADBEEF)
	binary.LittleEndian.PutUint32(data[4:8], 0x00000001)

	result, _ := decodeResponse(data)
	if result != nil {
		t.Errorf("expected nil for unknown constructor, got %T", result)
	}
}

func TestDecodeResponse_Vector(t *testing.T) {
	// Build a vector response containing boolTrue elements.
	var buf bin.Buffer
	buf.PutID(bin.TypeVector)
	buf.PutInt(2) // count = 2

	bt := &tg.BoolTrue{}
	bt.Encode(&buf)
	bf := &tg.BoolFalse{}
	bf.Encode(&buf)

	result, err := decodeResponse(buf.Buf)
	if result == nil {
		t.Fatalf("decodeResponse returned nil for vector: %v", err)
	}

	vec, ok := result.(*genericVector)
	if !ok {
		t.Fatalf("expected *genericVector, got %T", result)
	}

	if len(vec.Elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(vec.Elems))
	}

	if _, ok := vec.Elems[0].(*tg.BoolTrue); !ok {
		t.Errorf("element 0: expected *tg.BoolTrue, got %T", vec.Elems[0])
	}
	if _, ok := vec.Elems[1].(*tg.BoolFalse); !ok {
		t.Errorf("element 1: expected *tg.BoolFalse, got %T", vec.Elems[1])
	}
}

func TestDecodeResponse_VectorOfUsers(t *testing.T) {
	// Build a vector with a user element.
	user := &tg.User{
		Self:      true,
		Bot:       true,
		ID:        123,
		AccessHash: 456,
		FirstName: "Test",
		Username:  "testbot",
	}

	var userBuf bin.Buffer
	if err := user.Encode(&userBuf); err != nil {
		t.Fatalf("encode user: %v", err)
	}

	var buf bin.Buffer
	buf.PutID(bin.TypeVector)
	buf.PutInt(1) // count = 1
	buf.Buf = append(buf.Buf, userBuf.Buf...)

	result, err := decodeResponse(buf.Buf)
	if result == nil {
		t.Fatalf("decodeResponse returned nil for user vector: %v", err)
	}

	vec, ok := result.(*genericVector)
	if !ok {
		t.Fatalf("expected *genericVector, got %T", result)
	}

	if len(vec.Elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(vec.Elems))
	}

	decoded, ok := vec.Elems[0].(*tg.User)
	if !ok {
		t.Fatalf("expected *tg.User, got %T", vec.Elems[0])
	}

	if decoded.ID != 123 {
		t.Errorf("user ID: got %d, want 123", decoded.ID)
	}
	if decoded.FirstName != "Test" {
		t.Errorf("user FirstName: got %q, want %q", decoded.FirstName, "Test")
	}
}

func TestCaptureDecoder(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	buf := &bin.Buffer{Buf: data}

	var c captureDecoder
	if err := c.Decode(buf); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if len(c.data) != 4 {
		t.Errorf("captured %d bytes, want 4", len(c.data))
	}
}

func TestGenericVectorString(t *testing.T) {
	vec := &genericVector{
		Elems: []interface{}{
			&tg.BoolTrue{},
			&tg.BoolFalse{},
		},
	}

	s := vec.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}

