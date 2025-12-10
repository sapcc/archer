// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package as3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplication_MarshalJSON(t *testing.T) {
	type fields struct {
		Class    string
		Label    string
		Remark   string
		Template string
		Services map[string]any
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "TestMarshalling",
			fields: fields{
				Class:    "TestClass",
				Label:    "TestLabel",
				Remark:   "TestRemark",
				Template: "none",
				Services: map[string]any{"TestService": Service{
					Label:               "TestServiceLabel",
					Remark:              "TestServiceRemark",
					Class:               "TestClass",
					AllowVlans:          nil,
					IRules:              nil,
					Mirroring:           "",
					PersistenceMethods:  nil,
					Pool:                Pointer{},
					ProfileL4:           nil,
					ProfileTCP:          nil,
					Snat:                nil,
					VirtualAddresses:    nil,
					TranslateServerPort: false,
					VirtualPort:         1234,
				}},
			},
			want:    []byte(`{"TestService":{"label":"TestServiceLabel","remark":"TestServiceRemark","class":"TestClass","allowVlans":null,"iRules":null,"mirroring":"","persistenceMethods":null,"pool":{},"virtualAddresses":null,"translateServerPort":false,"virtualPort":1234},"class":"TestClass","label":"TestLabel","remark":"TestRemark","template":"none"}`),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Application{
				Class:    tt.fields.Class,
				Label:    tt.fields.Label,
				Remark:   tt.fields.Remark,
				Template: tt.fields.Template,
				Services: tt.fields.Services,
			}
			got, err := a.MarshalJSON()
			if !tt.wantErr(t, err, "MarshalJSON()") {
				return
			}
			assert.Equalf(t, tt.want, got, "MarshalJSON()")
		})
	}
}
