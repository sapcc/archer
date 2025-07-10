// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package f5os

type OpenconfigPlatformState struct {
	OpenconfigPlatformState struct {
		Description                  string `json:"description"`
		SerialNo                     string `json:"serial-no"`
		PartNo                       string `json:"part-no"`
		Empty                        bool   `json:"empty"`
		F5PlatformTpmIntegrityStatus string `json:"f5-platform:tpm-integrity-status"`
		F5PlatformMemory             struct {
			Total               string `json:"total"`
			Available           string `json:"available"`
			Free                string `json:"free"`
			UsedPercent         int    `json:"used-percent"`
			PlatformTotal       string `json:"platform-total"`
			PlatformUsed        string `json:"platform-used"`
			PlatformUsedPercent int    `json:"platform-used-percent"`
		} `json:"f5-platform:memory"`
		F5PlatformFileSystems struct {
			FileSystem []struct {
				Area        string `json:"area"`
				Category    string `json:"category"`
				Total       string `json:"total"`
				Free        string `json:"free"`
				Used        string `json:"used"`
				UsedPercent int    `json:"used-percent"`
			} `json:"file-system"`
		} `json:"f5-platform:file-systems"`
		F5PlatformTemperature struct {
			Current string `json:"current"`
			Average string `json:"average"`
			Minimum string `json:"minimum"`
			Maximum string `json:"maximum"`
		} `json:"f5-platform:temperature"`
	} `json:"openconfig-platform:state"`
}
