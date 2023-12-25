package handlers

import (
	"bytes"
	"encoding/json"
	"io"
)

func unmarshalBody[T interface{}](body io.ReadCloser, parsedBody *T) error {
	var buf bytes.Buffer

	_, err := buf.ReadFrom(body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(buf.Bytes(), &parsedBody); err != nil {
		return err
	}

	return nil
}
