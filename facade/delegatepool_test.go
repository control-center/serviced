// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"github.com/control-center/serviced/domain/govpool"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/utils"
	. "gopkg.in/check.v1"
)

func (ft *FacadeTest) Test_AddGovernedPool(t *C) {
	poolID := "Test_AddGovernedPool"

	// bad message
	err := ft.Facade.AddGovernedPool(ft.CTX, poolID, "dummymsg")
	t.Assert(err, NotNil)

	packet := utils.PacketData{"remotepoolid", "remoteaddress", "dummysecret"}
	msg, err := utils.EncodePacket(packet)
	if err != nil {
		t.Fatalf("Could not create packet: %s", err)
	}

	// pool does not exist
	err = ft.Facade.AddGovernedPool(ft.CTX, poolID, msg)
	t.Assert(err, Equals, ErrPoolNotExists)

	// success
	err = ft.Facade.AddResourcePool(ft.CTX, &pool.ResourcePool{ID: poolID})
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveResourcePool(ft.CTX, poolID)
	err = ft.Facade.AddGovernedPool(ft.CTX, poolID, msg)
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveGovernedPool(ft.CTX, poolID)

	// verify gov data
	gpool, err := ft.Facade.GetGovernedPool(ft.CTX, packet.RemotePoolID)
	t.Assert(err, IsNil)
	t.Assert(gpool, NotNil)
	t.Check(gpool.PoolID, Equals, poolID)
	t.Check(gpool.RemotePoolID, Equals, packet.RemotePoolID)
	t.Check(gpool.RemoteAddress, Equals, packet.RemoteAddress)
	gpool, err = ft.Facade.GetGovernedPoolByPoolID(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(gpool, NotNil)
	t.Check(gpool.PoolID, Equals, poolID)
	t.Check(gpool.RemotePoolID, Equals, packet.RemotePoolID)
	t.Check(gpool.RemoteAddress, Equals, packet.RemoteAddress)

	// TODO: verify secret
	// secret, err := ft.Facade.getPoolSecret(ft.CTX, packet.RemotePoolID)
	// t.Assert(err, IsNil)
	// t.Check(secret, Equals, packet.Secret)

	// add another gov to the same pool
	packet.RemotePoolID = "remotepoolid2"
	packet.RemoteAddress = "remoteaddress2"
	msg, err = utils.EncodePacket(packet)
	if err != nil {
		t.Fatalf("Could not create packet: %s", err)
	}
	err = ft.Facade.AddGovernedPool(ft.CTX, poolID, msg)
	t.Assert(err, Equals, ErrGovPoolExists)
}

func (ft *FacadeTest) Test_RemoveGovernedPool(t *C) {
	poolID := "Test_RemoteGovernedPool"
	packet := utils.PacketData{"remotepoolid", "remoteaddress", "dummysecret"}
	msg, err := utils.EncodePacket(packet)
	if err != nil {
		t.Fatalf("Could not create packet: %s", err)
	}

	err = ft.Facade.AddResourcePool(ft.CTX, &pool.ResourcePool{ID: poolID})
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveResourcePool(ft.CTX, poolID)

	err = ft.Facade.AddGovernedPool(ft.CTX, poolID, msg)
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveGovernedPool(ft.CTX, poolID)

	gpool, err := ft.Facade.GetGovernedPool(ft.CTX, packet.RemotePoolID)
	t.Assert(err, IsNil)
	t.Assert(gpool, NotNil)
	t.Check(gpool.PoolID, Equals, poolID)
	t.Check(gpool.RemotePoolID, Equals, packet.RemotePoolID)
	t.Check(gpool.RemoteAddress, Equals, packet.RemoteAddress)

	err = ft.Facade.RemoveGovernedPool(ft.CTX, poolID)
	t.Assert(err, IsNil)

	gpool, err = ft.Facade.GetGovernedPool(ft.CTX, packet.RemotePoolID)
	t.Assert(err, IsNil)
	t.Assert(gpool, IsNil)
}

func (ft *FacadeTest) Test_GetGovernedPool(t *C) {
	poolID := "Test_GetGovernedPool"
	packet := utils.PacketData{"remotepoolid", "remoteaddress", "dummysecret"}
	msg, err := utils.EncodePacket(packet)
	if err != nil {
		t.Fatalf("Could not create packet: %s", err)
	}

	err = ft.Facade.AddResourcePool(ft.CTX, &pool.ResourcePool{ID: poolID})
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveResourcePool(ft.CTX, poolID)

	err = ft.Facade.AddGovernedPool(ft.CTX, poolID, msg)
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveGovernedPool(ft.CTX, poolID)

	gpool, err := ft.Facade.GetGovernedPool(ft.CTX, packet.RemotePoolID)
	t.Assert(err, IsNil)
	t.Assert(gpool, NotNil)
	t.Check(gpool.PoolID, Equals, poolID)
	t.Check(gpool.RemotePoolID, Equals, packet.RemotePoolID)
	t.Check(gpool.RemoteAddress, Equals, packet.RemoteAddress)
}

func (ft *FacadeTest) Test_GetGovernedPoolByPoolID(t *C) {
	poolID := "Test_GetGovernedPoolByPoolID"
	packet := utils.PacketData{"remotepoolid", "remoteaddress", "dummysecret"}
	msg, err := utils.EncodePacket(packet)
	if err != nil {
		t.Fatalf("Could not create packet: %s", err)
	}

	err = ft.Facade.AddResourcePool(ft.CTX, &pool.ResourcePool{ID: poolID})
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveResourcePool(ft.CTX, poolID)

	err = ft.Facade.AddGovernedPool(ft.CTX, poolID, msg)
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveGovernedPool(ft.CTX, poolID)

	gpool, err := ft.Facade.GetGovernedPoolByPoolID(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(gpool, NotNil)
	t.Check(gpool.PoolID, Equals, poolID)
	t.Check(gpool.RemotePoolID, Equals, packet.RemotePoolID)
	t.Check(gpool.RemoteAddress, Equals, packet.RemoteAddress)
}

func (ft *FacadeTest) Test_GetGovernedPools(t *C) {
	expectedpools := []govpool.GovernedPool{}
	secret := "fauxsecret"

	// Zero pools
	actualpools, err := ft.Facade.GetGovernedPools(ft.CTX)
	t.Assert(err, IsNil)
	t.Assert(actualpools, DeepEquals, expectedpools)

	// One pool
	gpool := govpool.GovernedPool{
		PoolID:        "Test_GetGovernedPools_1",
		RemotePoolID:  "test_remote_1",
		RemoteAddress: "test_address_1",
	}
	err = ft.Facade.AddResourcePool(ft.CTX, &pool.ResourcePool{ID: gpool.PoolID})
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveResourcePool(ft.CTX, gpool.PoolID)
	msg, err := utils.EncodePacket(utils.PacketData{gpool.RemotePoolID, gpool.RemoteAddress, secret})
	t.Assert(err, IsNil)
	err = ft.Facade.AddGovernedPool(ft.CTX, gpool.PoolID, msg)
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveGovernedPool(ft.CTX, gpool.PoolID)
	gpool.DatabaseVersion++
	expectedpools = append(expectedpools, gpool)
	actualpools, err = ft.Facade.GetGovernedPools(ft.CTX)
	t.Assert(err, IsNil)
	t.Assert(actualpools, DeepEquals, expectedpools)

	// Two pools
	gpool = govpool.GovernedPool{
		PoolID:        "Test_GetGovernedPools_2",
		RemotePoolID:  "test_remote_2",
		RemoteAddress: "test_address_2",
	}
	err = ft.Facade.AddResourcePool(ft.CTX, &pool.ResourcePool{ID: gpool.PoolID})
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveResourcePool(ft.CTX, gpool.PoolID)
	msg, err = utils.EncodePacket(utils.PacketData{gpool.RemotePoolID, gpool.RemoteAddress, secret})
	t.Assert(err, IsNil)
	err = ft.Facade.AddGovernedPool(ft.CTX, gpool.PoolID, msg)
	t.Assert(err, IsNil)
	defer ft.Facade.RemoveGovernedPool(ft.CTX, gpool.PoolID)
	gpool.DatabaseVersion++
	expectedpools = append(expectedpools, gpool)
	actualpools, err = ft.Facade.GetGovernedPools(ft.CTX)
	t.Assert(err, IsNil)
	t.Assert(actualpools, DeepEquals, expectedpools)
}

func (ft *FacadeTest) Test_addPoolSecret(t *C) {
	t.Skip("skip this test until this is implemented")
	poolID := "Test_addPoolSecret"
	secret := "test_pool_secret"
	err := ft.Facade.addPoolSecret(ft.CTX, poolID, secret)
	t.Assert(err, IsNil)
	defer ft.Facade.removePoolSecret(ft.CTX, poolID)
	s, err := ft.Facade.getPoolSecret(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(s, Equals, secret)
	err = ft.Facade.addPoolSecret(ft.CTX, poolID, secret)
	t.Assert(err, Equals, ErrRemotePoolExists)
}

func (ft *FacadeTest) Test_removePoolSecret(t *C) {
	t.Skip("skip this test until this is implemented")
	poolID := "Test_removePoolSecret"
	secret := "test_pool_secret"
	err := ft.Facade.addPoolSecret(ft.CTX, poolID, secret)
	t.Assert(err, IsNil)
	defer ft.Facade.removePoolSecret(ft.CTX, poolID)
	s, err := ft.Facade.getPoolSecret(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(s, Equals, secret)
	err = ft.Facade.removePoolSecret(ft.CTX, poolID)
	t.Assert(err, IsNil)
	s, err = ft.Facade.getPoolSecret(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(s, Equals, "")
}

func (ft *FacadeTest) Test_getPoolSecret(t *C) {
	t.Skip("skip this test until this is implemented")
	poolID := "Test_getPoolSecret"
	secret := "test_pool_secret"
	err := ft.Facade.addPoolSecret(ft.CTX, poolID, secret)
	t.Assert(err, IsNil)
	defer ft.Facade.removePoolSecret(ft.CTX, poolID)
	s, err := ft.Facade.getPoolSecret(ft.CTX, poolID)
	t.Assert(err, IsNil)
	t.Assert(s, Equals, secret)
}