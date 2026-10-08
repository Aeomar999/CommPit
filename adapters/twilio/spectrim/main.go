// Command spectrim trims a full twilio-oai provider spec down to the
// endpoints mocksms implements, keeping the transitive component closure.
//
// It exists so the pinned copies under adapters/twilio/spec stay small and
// reviewable while the refresh workflow can still re-derive them
// deterministically from upstream:
//
//	spectrim -in twilio_api_v2010.json -out spec/twilio_api_v2010.json \
//	  -paths /2010-04-01/Accounts/{AccountSid}/Messages.json,/2010-04-01/Accounts/{AccountSid}/Messages/{Sid}.json
//
// Output is canonical JSON (indented, sorted keys) so re-runs are stable.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	in := flag.String("in", "", "full upstream spec file (JSON)")
	out := flag.String("out", "", "trimmed spec output file (JSON)")
	paths := flag.String("paths", "", "comma-separated exact spec paths to keep")
	flag.Parse()

	if *in == "" || *out == "" || *paths == "" {
		fmt.Fprintln(os.Stderr, "usage: spectrim -in full.json -out trimmed.json -paths p1,p2")
		os.Exit(2)
	}

	raw, err := os.ReadFile(*in)
	if err != nil {
		fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		fatal(err)
	}

	pathsNode, ok := doc["paths"].(map[string]any)
	if !ok {
		fatal(fmt.Errorf("spec has no paths object"))
	}
	keptPaths := map[string]any{}
	for _, p := range strings.Split(*paths, ",") {
		p = strings.TrimSpace(p)
		node, ok := pathsNode[p]
		if !ok {
			fatal(fmt.Errorf("path %q not found in spec", p))
		}
		keptPaths[p] = node
	}

	// Collect the transitive "#/components/<group>/<name>" closure.
	needed := map[string]map[string]bool{}
	var collect func(node any)
	collect = func(node any) {
		switch n := node.(type) {
		case map[string]any:
			for k, v := range n {
				if k == "$ref" {
					if s, ok := v.(string); ok {
						rest, found := strings.CutPrefix(s, "#/components/")
						if !found {
							continue
						}
						group, name, ok := strings.Cut(rest, "/")
						if !ok {
							continue
						}
						if needed[group] == nil {
							needed[group] = map[string]bool{}
						}
						needed[group][name] = true
					}
				} else {
					collect(v)
				}
			}
		case []any:
			for _, v := range n {
				collect(v)
			}
		}
	}
	for _, node := range keptPaths {
		collect(node)
	}
	components, _ := doc["components"].(map[string]any)
	keptComponents := map[string]any{}
	for changed := true; changed; {
		changed = false
		for group, names := range needed {
			groupNode, ok := components[group].(map[string]any)
			if !ok {
				fatal(fmt.Errorf("components group %q not found in spec", group))
			}
			kept, ok := keptComponents[group].(map[string]any)
			if !ok {
				kept = map[string]any{}
				keptComponents[group] = kept
			}
			for name := range names {
				if _, done := kept[name]; done {
					continue
				}
				node, ok := groupNode[name]
				if !ok {
					fatal(fmt.Errorf("component %s/%s not found in spec", group, name))
				}
				kept[name] = node
				before := countNeeded(needed)
				collect(node)
				if countNeeded(needed) != before {
					changed = true
				}
			}
		}
	}

	trimmed := map[string]any{}
	if v, ok := doc["openapi"]; ok {
		trimmed["openapi"] = v
	}
	if v, ok := doc["info"]; ok {
		trimmed["info"] = v
	}
	trimmed["paths"] = keptPaths
	if len(keptComponents) > 0 {
		trimmed["components"] = keptComponents
	}
	stripExamples(trimmed, false)

	encoded, err := json.MarshalIndent(trimmed, "", "  ")
	if err != nil {
		fatal(err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(*out, encoded, 0644); err != nil {
		fatal(err)
	}
}

func countNeeded(needed map[string]map[string]bool) int {
	total := 0
	for _, names := range needed {
		total += len(names)
	}
	return total
}

// stripExamples removes example/examples payloads from the trimmed spec.
// Upstream twilio-oai does not keep them valid (e.g. a non-RFC3339
// date-time example on the Messages list operation), and strict spec
// validation rejects the document because of them. Keys nested under a
// schema's "properties" are left alone, so a future property literally
// named "example" would survive.
func stripExamples(node any, underProperties bool) {
	switch n := node.(type) {
	case map[string]any:
		for k, v := range n {
			if (k == "example" || k == "examples") && !underProperties {
				delete(n, k)
				continue
			}
			stripExamples(v, k == "properties")
		}
	case []any:
		for _, v := range n {
			stripExamples(v, underProperties)
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "spectrim:", err)
	os.Exit(1)
}
