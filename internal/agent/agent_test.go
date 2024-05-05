package agent_test

import (
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/kawabatas/toy-log-distributed-system/internal/agent"
	"github.com/kawabatas/toy-log-distributed-system/internal/client"
	"github.com/kawabatas/toy-log-distributed-system/internal/server"
	"github.com/stretchr/testify/require"
)

func TestAgent(t *testing.T) {
	var agents []*agent.Agent
	for i := 0; i < 3; i++ {
		port0 := rand.Int31n(65_535-10_000) + 10_000
		port1 := rand.Int31n(65_535-10_000) + 10_000
		bindAddr := fmt.Sprintf("%s:%d", "127.0.0.1", port0)
		rpcPort := port1

		var startJoinAddrs []string
		if i != 0 {
			startJoinAddrs = append(startJoinAddrs, agents[0].Config.BindAddr)
		}

		agent, err := agent.New(agent.Config{
			NodeName:       fmt.Sprintf("%d", i),
			StartJoinAddrs: startJoinAddrs,
			BindAddr:       bindAddr,
			RPCPort:        int(rpcPort),
		})
		require.NoError(t, err)

		agents = append(agents, agent)
	}
	defer func() {
		for _, agent := range agents {
			err := agent.Shutdown()
			require.NoError(t, err)
		}
	}()
	time.Sleep(3 * time.Second)

	leaderClient, err := client.NewClient()
	require.NoError(t, err)
	addr0, _ := agents[0].RPCAddr()
	log.Printf("[DEBUG] addr0 %s\n", addr0)
	produceResponse, err := leaderClient.Produce(addr0, server.Record{Value: "foo"})
	require.NoError(t, err)
	consumeResponse, err := leaderClient.Consume(addr0, produceResponse)
	require.NoError(t, err)
	require.Equal(t, "foo", consumeResponse.Value)

	// レプリケーションが完了するまで待つ
	time.Sleep(3 * time.Second)

	followerClient, err := client.NewClient()
	require.NoError(t, err)
	addr1, _ := agents[1].RPCAddr()
	log.Printf("[DEBUG] addr1 %s\n", addr1)
	consumeResponse, err = followerClient.Consume(addr1, produceResponse)
	require.NoError(t, err)
	require.Equal(t, "foo", consumeResponse.Value)
}
