/*
 * Copyright contributors to the Hyperledger Fabric Operator project
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 * 	  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package version

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// DEPRECATED: operator versions for IBP logic
	V210        = "2.1.0"
	V212        = "2.1.2"
	V213        = "2.1.3"
	V250        = "2.5.0"
	V251        = "2.5.1"
	V252        = "2.5.2"
	V253        = "2.5.3"
	IBPOperator = "2.5.3"

	// IBM Support for Hyperledger Fabric product version
	V100     = "1.0.0"
	Operator = "1.0.0"
)

type String string

func (s String) Equal(new string) bool {
	oldVersion := newVersion(string(s))
	newVersion := newVersion(new)

	return oldVersion.equal(newVersion)

}
func (s String) EqualWithoutTag(new string) bool {
	oldVersion := newVersion(string(s))
	newVersion := newVersion(new)

	return oldVersion.equalWithoutTag(newVersion)

}

func (s String) GreaterThan(new string) bool {
	oldVersion := newVersion(string(s))
	newVersion := newVersion(new)

	return oldVersion.greaterThan(newVersion)
}

func (s String) LessThan(new string) bool {
	oldVersion := newVersion(string(s))
	newVersion := newVersion(new)

	return oldVersion.lessThan(newVersion)

}

type Version struct {
	Major   int `json:"major"`
	Minor   int `json:"minor"`
	Fixpack int `json:"fixpack"`
	Tag     int `json:"tag"`
}

func GetMajorReleaseVersion(version string) string {
	version = stripVersionPrefix(version)
	v := newVersion(version)
	switch v.Major {
	case 2:
		return V2
	case 1:
		return V1
	default:
		return V1
	}
}

func stripVersionPrefix(version string) string {
	return strings.TrimPrefix(strings.ToLower(version), "v")
}

func newVersion(version string) *Version {
	v := stringToIntList(version)

	switch len(v) {
	case 1:
		return &Version{
			Major: v[0],
		}
	case 2:
		return &Version{
			Major: v[0],
			Minor: v[1],
		}
	case 3:
		return &Version{
			Major:   v[0],
			Minor:   v[1],
			Fixpack: v[2],
		}
	case 4:
		return &Version{
			Major:   v[0],
			Minor:   v[1],
			Fixpack: v[2],
			Tag:     v[3],
		}
	}

	return &Version{}
}

func stringToIntList(version string) []int {
	var tag string

	// If version of format major.minor.fixpack-tag, extract tag first
	if strings.Contains(version, "-") {
		vList := strings.Split(version, "-")
		version = vList[0] // major.minor.fixpack
		tag = vList[1]     // tag
	}

	strList := strings.Split(version, ".")
	if tag != "" {
		strList = append(strList, tag)
	}

	intList := []int{}
	for _, str := range strList {
		num, err := strconv.Atoi(str)
		if err != nil {
			// No-op: strconv.Atoi() returns 0 with the error
		}
		intList = append(intList, num)
	}

	return intList
}

func (v *Version) equal(newVersion *Version) bool {
	if newVersion == nil {
		return false
	}

	if v.Major == newVersion.Major {
		if v.Minor == newVersion.Minor {
			if v.Fixpack == newVersion.Fixpack {
				if v.Tag == newVersion.Tag {
					return true
				}
			}
		}
	}

	return false
}

func (v *Version) equalWithoutTag(newVersion *Version) bool {
	if newVersion == nil {
		return false
	}

	if v.Major == newVersion.Major {
		if v.Minor == newVersion.Minor {
			if v.Fixpack == newVersion.Fixpack {
				return true
			}
		}
	}

	return false
}

func (v *Version) lessThan(newVersion *Version) bool {
	if v.Major < newVersion.Major {
		return true
	} else if v.Major > newVersion.Major {
		return false
	}

	if v.Minor < newVersion.Minor {
		return true
	} else if v.Minor > newVersion.Minor {
		return false
	}

	if v.Fixpack < newVersion.Fixpack {
		return true
	} else if v.Fixpack > newVersion.Fixpack {
		return false
	}

	if v.Tag < newVersion.Tag {
		return true
	}

	return false
}

func (v *Version) greaterThan(newVersion *Version) bool {
	if v.Major > newVersion.Major {
		return true
	} else if v.Major < newVersion.Major {
		return false
	}

	if v.Minor > newVersion.Minor {
		return true
	} else if v.Minor < newVersion.Minor {
		return false
	}

	if v.Fixpack > newVersion.Fixpack {
		return true
	} else if v.Fixpack < newVersion.Fixpack {
		return false
	}

	if v.Tag > newVersion.Tag {
		return true
	}

	return false
}

func (v *Version) String() string {
	if v != nil {
		return fmt.Sprintf("%d.%d.%d-%d", v.Major, v.Minor, v.Fixpack, v.Tag)
	}
	return "nil"
}
