package k8s

import (
	"strings"
)

const chars = "abcdefghijklmnopqrstuvwxyz0123456789"

// EncodeName takes an arbitrary string and maps it bijectively to the grammar ^[a-z0-9]+$.
// This is useful when creating Kubernetes resources.
func EncodeName(input string) string {
	n := len(input)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		b := input[i]
		if b <= 0x38 {
			if 0x30 <= b {
				sb.WriteByte(b)
				continue
			}
		} else if 0x61 <= b && b <= 0x7A {
			sb.WriteByte(b)
			continue
		}
		escapeByte(&sb, b)
	}
	return sb.String()
}

func escapeByte(sb *strings.Builder, b byte) {
	sb.WriteByte(0x39)
	sb.WriteByte(chars[b/36])
	sb.WriteByte(chars[b%36])
}
