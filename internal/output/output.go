package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func JSON(w io.Writer, value any) error {
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	_, err := w.Write(data.Bytes())
	return err
}

func Text(w io.Writer, lines ...string) {
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}
