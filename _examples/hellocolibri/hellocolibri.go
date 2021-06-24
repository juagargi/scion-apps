// Copyright 2021 ETH Zurich
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/scionproto/scion/go/lib/addr"
	libcol "github.com/scionproto/scion/go/lib/colibri"
	"github.com/scionproto/scion/go/lib/sciond"
)

const (
	sciondPath = "127.0.0.20:30255"
)

func main() {
	fmt.Println("started")
	// ctx, cancelF := context.WithTimeout(context.Background(), 10*time.Second)
	ctx, cancelF := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelF()

	sciond, err := sciond.NewService(sciondPath).Connect(ctx)
	check(err)
	// sciond.DRKeyGetLvl2Key
	dstIA, err := addr.IAFromString("1-ff00:0:112")
	check(err)

	stitchable, err := sciond.ColibriListRsvs(ctx, dstIA)
	check(err)
	fmt.Printf("received reservations to %s:\n%+v\n", dstIA, stitchable)

	trips := libcol.CombineAll(stitchable)
	fmt.Printf("Got %d trips\n", len(trips))
	for i, t := range trips {
		fmt.Printf("[%3d]: %s\n", i, t)
	}

	res, err := sciond.ColibriSetupRsv(ctx, trips[0])
}

// check just ensures the error is nil, or complains and quits
func check(e error) {
	if e != nil {
		panic(fmt.Sprintf("Fatal error: %v", e))
	}
}
