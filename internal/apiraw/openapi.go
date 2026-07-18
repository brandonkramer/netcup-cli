package apiraw

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

type Operation struct {
	Method      string
	Path        string
	Summary     string
	Description string
	Tag         string
	PathParams  []string
	QueryParams []string
	HasBody     bool
}

type Index struct {
	Ops []Operation
}

func Load(paths ...string) (*Index, error) {
	var last error
	for _, p := range paths {
		if p == "" {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			last = err
			continue
		}
		idx, err := Parse(b)
		if err != nil {
			return nil, err
		}
		return idx, nil
	}
	if last != nil {
		return nil, last
	}
	return nil, fmt.Errorf("no openapi document found")
}

func Parse(b []byte) (*Index, error) {
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	paths, _ := doc["paths"].(map[string]any)
	var ops []Operation
	for path, item := range paths {
		mops, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for method, raw := range mops {
			ml := strings.ToUpper(method)
			switch ml {
			case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD":
			default:
				continue
			}
			om, _ := raw.(map[string]any)
			summary, _ := om["summary"].(string)
			desc, _ := om["description"].(string)
			tag := ""
			if tags, ok := om["tags"].([]any); ok && len(tags) > 0 {
				tag, _ = tags[0].(string)
			}
			op := Operation{
				Method:      ml,
				Path:        path,
				Summary:     summary,
				Description: desc,
				Tag:         tag,
				PathParams:  pathParamNames(path),
				HasBody:     om["requestBody"] != nil,
			}
			if params, ok := om["parameters"].([]any); ok {
				for _, p := range params {
					pm, _ := p.(map[string]any)
					if pm["in"] == "query" {
						if name, _ := pm["name"].(string); name != "" {
							op.QueryParams = append(op.QueryParams, name)
						}
					}
				}
			}
			ops = append(ops, op)
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path == ops[j].Path {
			return ops[i].Method < ops[j].Method
		}
		return ops[i].Path < ops[j].Path
	})
	return &Index{Ops: ops}, nil
}

var pathParamRe = regexp.MustCompile(`\{([^}]+)\}`)

func pathParamNames(path string) []string {
	matches := pathParamRe.FindAllStringSubmatch(path, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

type Match struct {
	Op     Operation
	Score  int
	Reason string
}

// Resolve finds the best operation for a hint like:
//
//	"GET /api/v1/servers"
//	"/api/v1/servers/{serverId}"
//	"servers list"
//	"snapshot create"
func (idx *Index) Resolve(hint string) ([]Match, error) {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return nil, fmt.Errorf("empty operation hint")
	}
	parts := strings.Fields(hint)
	method := ""
	pathHint := hint
	if len(parts) >= 2 && isMethod(parts[0]) {
		method = strings.ToUpper(parts[0])
		pathHint = strings.Join(parts[1:], " ")
	} else if len(parts) == 1 && isMethod(parts[0]) {
		method = strings.ToUpper(parts[0])
		pathHint = ""
	}

	var matches []Match
	hl := strings.ToLower(hint)
	pl := strings.ToLower(pathHint)

	for _, op := range idx.Ops {
		if method != "" && op.Method != method {
			continue
		}
		score := 0
		reason := ""
		ol := strings.ToLower(op.Path)
		sl := strings.ToLower(op.Summary)
		tl := strings.ToLower(op.Tag)
		combo := ol + " " + sl + " " + tl

		if pathHint != "" && op.Path == pathHint {
			score = 1000
			reason = "exact path"
		} else if pathHint != "" && ol == pl {
			score = 1000
			reason = "exact path"
		} else if pathHint != "" && strings.Contains(ol, strings.Trim(pl, "/")) {
			score = 400 + pathOverlap(ol, pl)
			reason = "path contains"
		} else if strings.Contains(combo, hl) {
			score = 200
			reason = "summary/tag/path"
		} else {
			// token overlap
			tokens := tokenize(hl)
			hit := 0
			for _, t := range tokens {
				if strings.Contains(combo, t) {
					hit++
				}
			}
			if hit == 0 {
				continue
			}
			score = 50 * hit
			reason = fmt.Sprintf("%d token hits", hit)
		}
		if method != "" {
			score += 50
		}
		matches = append(matches, Match{Op: op, Score: score, Reason: reason})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Op.Path < matches[j].Op.Path
		}
		return matches[i].Score > matches[j].Score
	})
	return matches, nil
}

func (idx *Index) Best(hint string) (*Operation, []Match, error) {
	matches, err := idx.Resolve(hint)
	if err != nil {
		return nil, nil, err
	}
	if len(matches) == 0 {
		return nil, nil, fmt.Errorf("no operation matches %q", hint)
	}
	if len(matches) > 1 && matches[0].Score == matches[1].Score {
		return nil, matches[:min(5, len(matches))], fmt.Errorf("ambiguous operation %q (%d ties)", hint, countTies(matches))
	}
	// require clear winner unless exact
	if len(matches) > 1 && matches[0].Score < 400 && matches[0].Score-matches[1].Score < 50 {
		return nil, matches[:min(5, len(matches))], fmt.Errorf("ambiguous operation %q; refine hint", hint)
	}
	return &matches[0].Op, matches, nil
}

func FillPath(template string, params map[string]string) (string, error) {
	out := template
	for _, name := range pathParamNames(template) {
		v, ok := params[name]
		if !ok || v == "" {
			// try camelCase / common aliases
			for k, val := range params {
				if strings.EqualFold(k, name) {
					v = val
					ok = true
					break
				}
			}
		}
		if !ok || v == "" {
			return "", fmt.Errorf("missing path param {%s}", name)
		}
		out = strings.ReplaceAll(out, "{"+name+"}", v)
	}
	if strings.Contains(out, "{") {
		return "", fmt.Errorf("unfilled path params in %s", out)
	}
	return out, nil
}

func isMethod(s string) bool {
	switch strings.ToUpper(s) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD":
		return true
	default:
		return false
	}
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "/", " ")
	s = strings.ReplaceAll(s, "{", " ")
	s = strings.ReplaceAll(s, "}", " ")
	s = strings.ReplaceAll(s, ":", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	parts := strings.Fields(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) < 2 || isMethod(p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func pathOverlap(a, b string) int {
	as := strings.Split(strings.Trim(a, "/"), "/")
	bs := strings.Split(strings.Trim(b, "/"), "/")
	n := 0
	for _, x := range as {
		for _, y := range bs {
			if x == y && x != "" && !strings.HasPrefix(x, "{") {
				n += 20
			}
		}
	}
	return n
}

func countTies(m []Match) int {
	if len(m) == 0 {
		return 0
	}
	n := 1
	for i := 1; i < len(m) && m[i].Score == m[0].Score; i++ {
		n++
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
