package querybuilder

import (
	"bytes"
	"fmt"
)

type builder struct {
	buf    bytes.Buffer
	params []interface{}
}

func (b *builder) bind(value interface{}) {
	b.params = append(b.params, value)
	fmt.Fprintf(&b.buf, "$%d", len(b.params))
}

func (b *builder) write(s string) {
	b.buf.WriteString(s)
}
