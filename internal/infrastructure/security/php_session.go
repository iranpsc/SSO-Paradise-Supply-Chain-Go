package security

import (
	"errors"
	"strconv"
)

// ParsePHPSession supports primitive PHP session arrays only. It never creates
// PHP objects, executes code, or accepts references/oversized nested values.
func ParsePHPSession(raw []byte) (map[string]any, error) {
	if len(raw) > 1<<20 {
		return nil, errors.New("session exceeds limit")
	}
	p := phpParser{raw: raw}
	value, err := p.value(0)
	if err != nil || p.pos != len(raw) {
		return nil, errors.New("unsupported PHP session")
	}
	session, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("session is not an array")
	}
	return session, nil
}

type phpParser struct {
	raw []byte
	pos int
}

func (p *phpParser) literal(s string) bool {
	if p.pos+len(s) > len(p.raw) || string(p.raw[p.pos:p.pos+len(s)]) != s {
		return false
	}
	p.pos += len(s)
	return true
}
func (p *phpParser) number(delimiter byte) (string, error) {
	start := p.pos
	for p.pos < len(p.raw) && p.raw[p.pos] != delimiter {
		p.pos++
	}
	if p.pos == len(p.raw) {
		return "", errors.New("unterminated PHP value")
	}
	result := string(p.raw[start:p.pos])
	p.pos++
	return result, nil
}
func (p *phpParser) value(depth int) (any, error) {
	invalid := errors.New("unsupported PHP session value")
	if depth > 32 || p.pos >= len(p.raw) {
		return nil, invalid
	}
	kind := p.raw[p.pos]
	p.pos++
	if kind == 'N' {
		if !p.literal(";") {
			return nil, invalid
		}
		return nil, nil
	}
	if !p.literal(":") {
		return nil, invalid
	}
	switch kind {
	case 'i', 'b':
		raw, err := p.number(';')
		if err != nil {
			return nil, err
		}
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, invalid
		}
		if kind == 'b' {
			if n != 0 && n != 1 {
				return nil, invalid
			}
			return n == 1, nil
		}
		return n, nil
	case 's':
		raw, err := p.number(':')
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > len(p.raw)-p.pos-3 || !p.literal("\"") {
			return nil, invalid
		}
		value := string(p.raw[p.pos : p.pos+n])
		p.pos += n
		if !p.literal("\";") {
			return nil, invalid
		}
		return value, nil
	case 'a':
		raw, err := p.number(':')
		if err != nil {
			return nil, err
		}
		count, err := strconv.Atoi(raw)
		if err != nil || count < 0 || count > 10000 || !p.literal("{") {
			return nil, invalid
		}
		result := map[string]any{}
		for i := 0; i < count; i++ {
			key, err := p.value(depth + 1)
			if err != nil {
				return nil, err
			}
			var name string
			switch k := key.(type) {
			case string:
				name = k
			case int64:
				name = strconv.FormatInt(k, 10)
			default:
				return nil, invalid
			}
			value, err := p.value(depth + 1)
			if err != nil {
				return nil, err
			}
			if _, exists := result[name]; exists {
				return nil, invalid
			}
			result[name] = value
		}
		if !p.literal("}") {
			return nil, invalid
		}
		return result, nil
	default:
		return nil, invalid
	}
}
