/*
Copyright 2012 Vimeo Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package promoter

import "math/rand"

type OneInTenPromoter struct{}

type DefaultPromoter struct{}

func (p *OneInTenPromoter) ShouldPromote(key string, data []byte, stats Stats) bool {
	if rand.Intn(10) == 0 {
		return true
	}
	return false
}

func (p *DefaultPromoter) ShouldPromote(key string, data []byte, stats Stats) bool {
	keyQPS := stats.KeyQPS
	if keyQPS >= stats.HCStats.LeastRecentQPS {
		return true
	}
	return false
}

// HCStats keeps track of the size, capacity, and coldest/hottest
// elements in the hot cache
type HCStats struct {
	MostRecentQPS  float64
	LeastRecentQPS float64
	HCSize         int64
	HCCapacity     int64
}

// Stats contains both the KeyQPS and a pointer to the galaxy-wide
// HCStats
type Stats struct {
	KeyQPS  float64
	HCStats *HCStats
}