package http2

import (
	"encoding/binary"
	"errors"
	"github.com/ugorji/go/codec"
	"reflect"
)

type sessionHandler struct {
	session *Session
}

func (h *sessionHandler) encodeChannel(v reflect.Value) ([]byte, error) {
	rc := v.Interface().(Channel)
	if rc.stream == nil {
		return nil, errors.New("bad type")
	}

	// Get stream identifier?
	streamId := rc.stream.Identifier()
	var buf [9]byte
	if rc.direction == In {
		buf[0] = 0x02 // Reverse direction
	} else if rc.direction == Out {
		buf[0] = 0x01 // Reverse direction
	} else {
		return nil, errors.New("Invalid direction")
	}
	written := binary.PutUvarint(buf[1:], uint64(streamId))
	if written > 4 {
		return nil, errors.New("wrote unexpected stream id size")
	}
	return buf[:(written + 1)], nil
}

func (h *sessionHandler) decodeChannel(v reflect.Value, b []byte) error {
	rc := v.Interface().(Channel)

	if b[0] == 0x01 {
		rc.direction = In
	} else if b[0] == 0x02 {
		rc.direction = Out
	} else {
		return errors.New("unexpected direction")
	}

	streamId, readN := binary.Uvarint(b[1:])
	if readN > 4 {
		return errors.New("read unexpected stream id size")
	}
	stream := h.session.conn.FindStream(uint32(streamId))
	if stream == nil {
		return errors.New("stream does not exist")
	}
	rc.stream = stream
	v.Set(reflect.ValueOf(rc))

	return nil
}

func (h *sessionHandler) encodeStream(v reflect.Value) ([]byte, error) {
	bs := v.Interface().(ByteStream)
	if bs.referenceId == "" {
		return nil, errors.New("bad type")
	}
	return []byte(bs.referenceId), nil
}

func (h *sessionHandler) decodeStream(v reflect.Value, b []byte) error {
	bs := h.session.GetByteStream(string(b))
	if bs != nil {
		v.Set(reflect.ValueOf(*bs))
	}

	return nil
}

func getMsgPackHandler(session *Session) *codec.MsgpackHandle {
	h := &sessionHandler{session: session}
	mh := &codec.MsgpackHandle{}
	err := mh.AddExt(reflect.TypeOf(Channel{}), 1, h.encodeChannel, h.decodeChannel)
	if err != nil {
		panic(err)
	}

	err = mh.AddExt(reflect.TypeOf(ByteStream{}), 2, h.encodeStream, h.decodeStream)
	if err != nil {
		panic(err)
	}

	return mh
}
