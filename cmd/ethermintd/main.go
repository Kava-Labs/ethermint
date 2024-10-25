// Copyright 2021 Evmos Foundation
// This file is part of Evmos' Ethermint library.
//
// The Ethermint library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Ethermint library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Ethermint library. If not, see https://github.com/evmos/ethermint/blob/main/LICENSE
package main

import (
	"fmt"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/evmos/ethermint/app"
	cmdcfg "github.com/evmos/ethermint/cmd/config"
	"os"
)

func main() {
	cmdcfg.SetupConfig()
	cmdcfg.RegisterDenoms()

	rootCmd, _ := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, EnvPrefix, app.DefaultNodeHome); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
		//switch e := err.(type) {
		//case server.ErrorCode:
		//	os.Exit(e.Code)

		//default:
		//	os.Exit(1)
		//}
	}
}
