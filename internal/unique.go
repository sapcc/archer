// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package internal

func Unique(s []string) []string {
	in := make(map[string]struct{})
	result := make([]string, 0)
	for _, str := range s {
		if _, ok := in[str]; !ok {
			in[str] = struct{}{}
			result = append(result, str)
		}
	}
	return result
}
