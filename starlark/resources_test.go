// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package starlark

import (
	"strings"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

func TestResourcesFunc(t *testing.T) {
	tests := []struct {
		name   string
		kwargs func(t *testing.T) []starlark.Tuple
		eval   func(t *testing.T, kwargs []starlark.Tuple)
	}{
		{
			name:   "empty kwargs",
			kwargs: func(t *testing.T) []starlark.Tuple { return nil },
			eval: func(t *testing.T, kwargs []starlark.Tuple) {
				_, err := resourcesFunc(&starlark.Thread{Name: "test"}, nil, nil, kwargs)
				if err == nil {
					t.Fatal("expected failure, but err == nil")
				}
			},
		},
		{
			name: "bad args",
			kwargs: func(t *testing.T) []starlark.Tuple {
				return []starlark.Tuple{[]starlark.Value{starlark.String("foo"), starlark.String("bar")}}
			},
			eval: func(t *testing.T, kwargs []starlark.Tuple) {
				_, err := resourcesFunc(&starlark.Thread{Name: "test"}, nil, nil, kwargs)
				if err == nil {
					t.Fatal("expected failure, but err == nil")
				}
			},
		},
		{
			name: "missing ssh_config",
			kwargs: func(t *testing.T) []starlark.Tuple {
				return []starlark.Tuple{[]starlark.Value{starlark.String("hosts"), starlark.String("foo.host.1")}}
			},
			eval: func(t *testing.T, kwargs []starlark.Tuple) {
				_, err := resourcesFunc(&starlark.Thread{Name: "test"}, nil, nil, kwargs)
				if err == nil {
					t.Fatal("expected failure, but err == nil")
				}
			},
		},
		{
			name: "host only",
			kwargs: func(t *testing.T) []starlark.Tuple {
				return []starlark.Tuple{
					[]starlark.Value{starlark.String("hosts"), starlark.NewList([]starlark.Value{starlark.String("foo.host.1")})},
				}
			},
			eval: func(t *testing.T, kwargs []starlark.Tuple) {
				thread := newTestThreadLocal(t)
				thread.SetLocal(identifiers.sshCfg, starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
					"username":         starlark.String("uname"),
					"private_key_path": starlark.String("path"),
				}))
				res, err := resourcesFunc(thread, nil, nil, kwargs)
				if err != nil {
					t.Fatal(err)
				}
				resources, ok := res.(*starlark.List)
				if !ok {
					t.Fatalf("unexpected type for resource: %T", resources)
				}

				expectedHosts := []string{"foo.host.1"}
				for i := 0; i < resources.Len(); i++ {
					resStruct, ok := resources.Index(i).(*starlarkstruct.Struct)
					if !ok {
						t.Fatalf("unexpected type for resource: %T", resources.Index(i))
					}

					val, err := resStruct.Attr("kind")
					if err != nil {
						t.Error(err)
					}
					if trimQuotes(val.String()) != identifiers.hostResource {
						t.Errorf("unexpected resource kind for host list provider")
					}

					transport, err := resStruct.Attr("transport")
					if err != nil {
						t.Error(err)
					}
					if trimQuotes(transport.String()) != "ssh" {
						t.Errorf("unexpected %s transport: %s", identifiers.resources, transport)
					}

					sshCfg, err := resStruct.Attr(identifiers.sshCfg)
					if err != nil {
						t.Error(err)
					}
					if sshCfg == nil {
						t.Error("resources missing ssh_config")
					}

					host, err := resStruct.Attr("host")
					if err != nil {
						t.Error(err)
					}

					if trimQuotes(host.String()) != expectedHosts[0] {
						t.Error("unexpected value for names list in resources")
					}
				}
			},
		},
		{
			name: "provider only",
			kwargs: func(t *testing.T) []starlark.Tuple {
				provider, err := hostListProvider(
					newTestThreadLocal(t),
					nil, nil,
					[]starlark.Tuple{
						[]starlark.Value{starlark.String("hosts"), starlark.NewList([]starlark.Value{
							starlark.String("local.host"),
							starlark.String("192.168.10.10"),
						})},
						[]starlark.Value{starlark.String("ssh_config"), starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
							"username":         starlark.String("uname"),
							"private_key_path": starlark.String("path"),
						})},
					},
				)

				if err != nil {
					t.Fatal(err)
				}

				return []starlark.Tuple{[]starlark.Value{starlark.String("provider"), provider}}
			},

			eval: func(t *testing.T, kwargs []starlark.Tuple) {
				res, err := resourcesFunc(newTestThreadLocal(t), nil, nil, kwargs)
				if err != nil {
					t.Fatal(err)
				}

				resources, ok := res.(*starlark.List)
				if !ok {
					t.Fatalf("unexpected type for resource: %T", resources)
				}

				expectedHosts := []string{"local.host", "192.168.10.10"}
				for i := 0; i < resources.Len(); i++ {
					resStruct, ok := resources.Index(i).(*starlarkstruct.Struct)
					if !ok {
						t.Fatalf("unexpected type for resource: %T", res)
					}
					val, err := resStruct.Attr("kind")
					if err != nil {
						t.Error(err)
					}
					if trimQuotes(val.String()) != identifiers.hostResource {
						t.Errorf("unexpected resource kind for host list provider")
					}

					transport, err := resStruct.Attr("transport")
					if err != nil {
						t.Error(err)
					}
					if trimQuotes(transport.String()) != "ssh" {
						t.Errorf("unexpected %s transport: %s", identifiers.resources, transport)
					}

					sshCfg, err := resStruct.Attr(identifiers.sshCfg)
					if err != nil {
						t.Error(err)
					}
					if sshCfg == nil {
						t.Error("resources missing ssh_config")
					}

					host, err := resStruct.Attr("host")
					if err != nil {
						t.Error(err)
					}
					if trimQuotes(host.String()) != expectedHosts[i] {
						t.Error("unexpected value for names list in resources")
					}
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.eval(t, test.kwargs(t))
		})
	}
}

func TestResourceScript(t *testing.T) {
	tests := []struct {
		name   string
		script string
		eval   func(t *testing.T, script string)
	}{
		{
			name: "resources assigned",
			script: `
set_defaults(ssh_config(username = "uname"))
res = resources(hosts=["foo.host.1", "local.host", "10.10.10.1"])`,
			eval: func(t *testing.T, script string) {
				exe := New()
				if err := exe.Exec("test.star", strings.NewReader(script)); err != nil {
					t.Fatal(err)
				}
				data := exe.result["res"]
				if data == nil {
					t.Fatalf("%s function call not returning value", identifiers.resources)
				}

				resources, ok := data.(*starlark.List)
				if !ok {
					t.Fatalf("expecting *starlark.Struct, got %T", data)
				}

				expectedHosts := []string{"foo.host.1", "local.host", "10.10.10.1"}
				for i := 0; i < resources.Len(); i++ {
					resStruct, ok := resources.Index(i).(*starlarkstruct.Struct)
					if !ok {
						t.Fatalf("expecting *starlark.Struct, got %T", resources.Index(i))
					}

					val, err := resStruct.Attr("kind")
					if err != nil {
						t.Error(err)
					}
					if trimQuotes(val.String()) != identifiers.hostResource {
						t.Errorf("unexpected resource kind for host list provider")
					}

					transport, err := resStruct.Attr("transport")
					if err != nil {
						t.Error(err)
					}
					if trimQuotes(transport.String()) != "ssh" {
						t.Errorf("unexpected %s transport: %s", identifiers.resources, transport)
					}

					sshCfg, err := resStruct.Attr(identifiers.sshCfg)
					if err != nil {
						t.Error(err)
					}
					if sshCfg == nil {
						t.Error("resources missing ssh_config")
					}

					host, err := resStruct.Attr("host")
					if err != nil {
						t.Error(err)
					}

					if trimQuotes(host.String()) != expectedHosts[i] {
						t.Error("unexpected value for names list in resources")
					}
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.eval(t, test.script)
		})
	}
}
