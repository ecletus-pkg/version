package version

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"encoding/base64"

	"crypto/sha1"

	"github.com/ecletus/cli"
	"github.com/ecletus/plug"
	"github.com/spf13/cobra"
)

type Attributes struct {
	CommitID   string     `json:",omitempty"`
	CommitDate *time.Time `json:",omitempty"`
	BuildDate  *time.Time `json:",omitempty"`
	HomePage   string     `json:",omitempty"`
	Hash       string     `json:",omitempty"`
}

func (a Attributes) MarshalJSONIndent(prefix, indent string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent(prefix, indent)
	if err := enc.Encode(&a); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type IndentMarshaller interface {
	MarshalJSONIndent(dprefix, indent string) ([]byte, error)
}

type marshalher struct {
	Do func() ([]byte, error)
}

func (m marshalher) MarshalJSON() ([]byte, error) {
	return m.Do()
}

func Indent(prefix, indent string, value IndentMarshaller) *marshalher {
	return &marshalher{Do: func() (i []byte, e error) {
		prefix := prefix
		return value.MarshalJSONIndent(prefix, indent)
	}}
}

type Version struct {
	Attributes
	Assets map[string]*Version
}

func (v Version) MarshalJSON() ([]byte, error) {
	var assets Versions
	if v.Assets != nil {
		for key, value := range v.Assets {
			assets = append(assets, NamedVersion{key, value})
		}
	}
	data := map[string]interface{}{
		"Attributes": &v.Attributes,
	}
	if len(assets) > 0 {
		data["Assets"] = assets
	}
	return json.Marshal(data)
}

func (v Version) MarshalJSONIndent(prefix, indent string) ([]byte, error) {
	var assets Versions
	if v.Assets != nil {
		for key, value := range v.Assets {
			assets = append(assets, NamedVersion{key, value})
		}
	}

	var (
		buf  bytes.Buffer
		err  error
		data []byte
	)
	enc := json.NewEncoder(&buf)
	enc.SetIndent(prefix, indent)

	buf.WriteString("{\n" + prefix + indent)
	buf.WriteString(`"Attributes": `)
	data, err = v.Attributes.MarshalJSONIndent(prefix+indent, indent)
	if err != nil {
		return nil, err
	}
	buf.Write(trimLastBlankLine(data))
	if len(assets) > 0 {
		buf.WriteString(",\n" + prefix + indent + `"Assets": `)
		data, err = assets.MarshalJSONIndent(prefix+indent, indent)
		if err != nil {
			return nil, err
		}
		buf.Write(trimLastBlankLine(data))
	}
	buf.WriteString("\n" + prefix + "}\n")
	return buf.Bytes(), nil
}

func (v Version) String() (r string) {
	var values []string
	if v.HomePage != "" {
		values = append(values, "HomePage: "+v.HomePage)
	}
	if v.CommitID != "" {
		values = append(values, "CommitID: "+v.CommitID)
	}
	if v.CommitDate != nil && !v.CommitDate.IsZero() {
		values = append(values, "CommitDate: "+v.CommitDate.Format(time.RFC3339))
	}
	if v.BuildDate != nil && !v.BuildDate.IsZero() {
		values = append(values, "BuildDate: "+v.BuildDate.Format(time.RFC3339))
	}
	if v.Hash != "" {
		values = append(values, "Hash: "+v.Hash)
	}
	if r = strings.Join(values, "\n\t"); r != "" {
		r = "\n\t" + r
	}
	return r
}

type Plugin struct {
	plug.EventDispatcher
	Version      Version
	Versions     Versions
	VersionsFunc []func() Versions
}

type NamedVersion struct {
	Name string
	*Version
}

type Versions []NamedVersion

func (vs Versions) MarshalJSONIndent(prefix, indent string) ([]byte, error) {
	sort.Slice(vs, func(i, j int) bool {
		return vs[i].Name < vs[j].Name
	})

	var buf bytes.Buffer

	buf.WriteString("{\n")

	if len(vs) > 0 {
		var last []byte
		for i, kv := range vs {
			if i != 0 {
				buf.Write(trimLastBlankLine(last))
				buf.WriteString(",\n")
			}
			buf.WriteString(prefix + indent + `"` + kv.Name + `": `)
			// marshal value
			val, err := kv.Version.MarshalJSONIndent(prefix+indent, indent)
			if err != nil {
				return nil, err
			}
			last = val
		}

		buf.Write(trimLastBlankLine(last))
		buf.WriteString("\n")
	}

	buf.WriteString(prefix + "}\n")
	return buf.Bytes(), nil
}

func (vs Versions) MarshalJSON() ([]byte, error) {
	sort.Slice(vs, func(i, j int) bool {
		return vs[i].Name < vs[j].Name
	})

	var buf bytes.Buffer

	buf.WriteString("{")
	for i, kv := range vs {
		if i != 0 {
			buf.WriteString(",")
		}
		// marshal key
		key, err := json.Marshal(kv.Name)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteString(":")
		// marshal value
		val, err := json.Marshal(kv.Version)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}

func (p *Plugin) OnRegister() {
	return
	cli.OnRegister(p, func(e *cli.RegisterEvent) {
		dis := e.PluginDispatcher()
		version := &cobra.Command{
			Use:   "version",
			Short: "Show version",
			RunE: func(cmd *cobra.Command, args []string) error {
				main := p.Version
				assets := map[string]*Version{}

				if main.Assets != nil {
					for k, v := range main.Assets {
						assets[k] = v
					}
				}
				main.Assets = assets

				for _, v := range p.Versions {
					main.Assets[v.Name] = v.Version
				}
				for _, f := range p.VersionsFunc {
					for _, v := range f() {
						main.Assets[v.Name] = v.Version
					}
				}
				triggerRegister(dis, &main.Assets)
				if data, err := main.MarshalJSONIndent("", "  "); err != nil {
					return err
				} else {
					os.Stdout.Write(data)
					return nil
				}
			},
		}

		e.RootCmd.AddCommand(version)
	})
}

func SetEnv(v Version) {
	b, _ := v.MarshalJSON()
	os.Setenv(envName, base64.RawURLEncoding.EncodeToString(b))
}

func FromEnv() *Version {
	s := os.Getenv(envName)
	if s == "" {
		return nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	v := &Version{}
	if err = json.Unmarshal(b, v); err != nil {
		panic(err)
	}
	return v
}

var envName = func() string {
	s := sha1.Sum([]byte(filepath.Base(os.Args[0])))
	return "github_com__ecletus_pkg__version__" + string(s[:])
}()
