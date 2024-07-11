// Copyright 2023 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errors

import (
	"errors"
)

var (
	ErrBadRequest       = errors.New("bad request")
	ErrProjectMismatch  = errors.New("project mismatch")
	ErrMissingIPAddress = errors.New("missing ip address")
	ErrMissingSubnets   = errors.New("network has no subnets")
	ErrNoSelfIP         = errors.New("no self ip found")
	ErrNoVCMPFound      = errors.New("no VCMP guest found")
	ErrQuotaExceeded    = errors.New("quota has been met")
	ErrNoPhysNetFound   = errors.New("no physical network found")
)
