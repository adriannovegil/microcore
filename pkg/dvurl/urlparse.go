package dvurl

import (
	"github.com/Dobryvechir/microcore/pkg/dvmeta"
	"github.com/Dobryvechir/microcore/pkg/dvparser"
	"log"
	"regexp"
	"strings"
)

const (
	MaxCharsForStar = 1000000000
)

func cutOffFixedParts(data string) (*dvmeta.MaskInfo, string) {
	res := &dvmeta.MaskInfo{Middle: make([]*dvmeta.MaskInfoPart, 0, 3)}
	n := len(data)
	i := 0
	for i = 0; i < n; i++ {
		c := data[i]
		if c <= ' ' {
			continue
		} else if c == '!' {
			res.IsNegative = true
		} else if c == '<' {
			res.IsCaseInsensitive = -MatchCaseInsensitive
		} else if c == '>' {
			res.IsCaseInsensitive = MatchCaseInsensitive
		} else {
			break
		}
	}
	if i > 0 {
		data = data[i:]
		n = len(data)
	}
	for n > 0 {
		c := data[n-1]
		if c <= ' ' || c == '/' {
			n--
		} else {
			break
		}
	}
	if n != len(data) {
		data = data[:n]
	}
	pos := 0
	for i = 0; i < n; i++ {
		c := data[i]
		if (c <= ' ' || c == '/') && i == pos {
			pos = i + 1
		} else if c == '/' {
			if data[i-1] == '/' {
				res.FixedStart += data[pos:i]
				pos = i + 1
			}
		} else if c == '?' || c == '*' || c == '{' || c == '`' || c == '^' {
			break
		}
	}
	if pos < i {
		res.FixedStart += data[pos:i]
	}
	if i > 0 {
		data = data[i:]
		n = len(data)
	}
	for i = n - 1; i >= 0; i-- {
		c := data[i]
		if c == '?' || c == '*' || c == '`' || c == '$' {
			break
		}
		if c == '/' && data[i+1] == '/' {
			res.FixedEnd = data[i+1:n] + res.FixedEnd
			for i >= 0 && data[i] == '/' {
				i--
			}
			i++
			n = i
		}
	}
	if i+1 < n {
		res.FixedEnd = data[i+1:n] + res.FixedEnd
	}
	if len(data) != i+1 {
		data = data[:i+1]
	}
	return res, data
}

func ConvertSingleMask(data string) (res *dvmeta.MaskInfo) {
	res, data = cutOffFixedParts(data)
	n := len(data)
	for i := 0; i < n; {
		c := data[i]
		switch c {
		case '?', '*':
			{
				min := 0
				max := 0
				kind := dvmeta.MaskSlashAware
				for ; i < n; i++ {
					c = data[i]
					if c == '?' {
						min++
						max++
					} else if c == '*' {
						if i > 0 && data[i-1] == '*' {
							kind = dvmeta.MaskSlashUnaware
						}
						max = MaxCharsForStar
					} else {
						break
					}
				}
				res.Middle = append(res.Middle, &dvmeta.MaskInfoPart{Min: min, Max: max, Kind: kind})
			}
		case '`', '^':
			{
				endStr := "`"
				exclude := 1
				if c == '^' {
					endStr = "$"
					exclude = 0
				}
				pos := strings.Index(data[i+1:], endStr)
				if pos < 0 {
					log.Printf("Expected closing %s in url %s at %d\n", endStr, res.FixedStart+data+res.FixedEnd, i+len(res.FixedStart))
					data += res.FixedEnd + endStr
					res.FixedEnd = ""
					pos = len(data) - 1
				} else {
					pos += i + 1
				}
				reg := data[i+exclude : pos+1-exclude]
				if reg != "" {
					regex, err := regexp.Compile(reg)
					if err != nil {
						log.Printf("Incorrect regular expression %s : %s\n", reg, err.Error())
						res.Middle = append(res.Middle, &dvmeta.MaskInfoPart{Kind: dvmeta.MaskWord, Data: "?? Error ??"})
					} else {
						res.Middle = append(res.Middle, &dvmeta.MaskInfoPart{Kind: dvmeta.MaskRegExp, Regex: regex, Min: 0, Max: MaxCharsForStar})
					}
				}
				i = pos + 1
			}
		case '{':
			{
				i++
				kind := dvmeta.MaskSlashAware
				endBlock := "}"
				if data[i] == '{' {
					kind = dvmeta.MaskCondition
					endBlock = "}}"
					i++
				}
				pos := strings.Index(data[i:], endBlock)
				if pos < 0 {
					log.Printf("Expected closing %s in url  %s at %d\n", endBlock, data, i)
					continue
				}
				pos += i
				word := data[i:pos]
				res.Middle = append(res.Middle, &dvmeta.MaskInfoPart{Min: 1,
					Max:       MaxCharsForStar,
					Kind:      kind,
					Data:      word,
					Condition: word,
				})
				i = pos + len(endBlock)
			}
		default:
			{
				pos := i
				for i++; i < n; i++ {
					c := data[i]
					if c == '?' || c == '*' || c == '{' || c == '`' || c == '^' {
						break
					}
				}
				res.Middle = append(res.Middle, &dvmeta.MaskInfoPart{Kind: dvmeta.MaskWord, Data: data[pos:i]})
			}
		}
	}
	return
}

func ScanNamedIds(maskInfo *dvmeta.MaskInfo) (res []string) {
	res = make([]string, 0, 2)
	if maskInfo != nil && maskInfo.Middle != nil {
		for _, v := range maskInfo.Middle {
			if v.Kind == dvmeta.MaskSlashAware && v.Data != "" {
				res = append(res, v.Data)
			}
		}
	}
	return
}

func PreparseMaskExpressions(data string) (masks []*dvmeta.MaskInfo) {
	parts := dvparser.ConvertToNonEmptyList(data)
	n := len(parts)
	masks = make([]*dvmeta.MaskInfo, n)
	for i := 0; i < n; i++ {
		masks[i] = ConvertSingleMask(parts[i])
	}
	return
}