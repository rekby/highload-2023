package examples

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rekby/fixenv"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func DockerHost(e Env) string {
	f := func() (*fixenv.GenericResult[string], error) {
		if host := os.Getenv("DOCKER_HOST"); host != "" {
			return fixenv.NewGenericResult(host), nil
		}

		var searchPaths = []string{client.DefaultDockerHost}
		if path, err := os.UserHomeDir(); err == nil {
			searchPaths = append(searchPaths, filepath.Join(path, ".colima/default/docker.sock"))
		}

		for _, path := range searchPaths {
			if stat, err := os.Stat(path); err == nil {
				if stat.Mode()&os.ModeSocket != 0 {
					path = "unix://" + path
					return fixenv.NewGenericResult(path), nil
				}
			}
		}

		return nil, errors.New("docker socket path not found")
	}
	return fixenv.CacheResult(e, f, fixenv.CacheOptions{Scope: fixenv.ScopePackage})
}

func DockerClient(e Env) *client.Client {
	f := func() (*fixenv.GenericResult[*client.Client], error) {
		c, err := client.NewClientWithOpts(
			client.FromEnv,
			client.WithHost(DockerHost(e)),
			client.WithAPIVersionNegotiation(),
		)
		clean := func() {
			if c != nil {
				_ = c.Close()
			}
		}
		return fixenv.NewGenericResultWithCleanup(c, clean), err
	}

	return fixenv.CacheResult(e, f, fixenv.CacheOptions{Scope: fixenv.ScopePackage})
}

func YDBDocker(e Env) string {
	f := func() (_ *fixenv.GenericResult[string], resErr error) {
		const label = "highload-2023-fixenv"
		const image = "cr.yandex/yc/yandex-docker-local-ydb:latest"
		ctx := context.Background()
		c := DockerClient(e)

		// remove prev containers
		containers, err := c.ContainerList(ctx, types.ContainerListOptions{All: true})
		if err != nil {
			return nil, fmt.Errorf("failed to list containers: %+v", err)
		}

		// cleanup from previous runs
		for _, dockerContainer := range containers {
			if _, ok := dockerContainer.Labels[label]; ok {
				e.T().Logf("removing container from previous start: %q", dockerContainer.ID)
			}
			err = c.ContainerRemove(ctx, dockerContainer.ID, types.ContainerRemoveOptions{Force: true})
			if err != nil {
				return nil, fmt.Errorf("failed to remove container %q: %+v", dockerContainer.ID, err)
			}
		}

		logResp := func(message string, res io.ReadCloser) {
			resBytes, err := io.ReadAll(res)
			_ = res.Close()
			if err != nil {
				e.T().Fatalf("failed to read image pull result: %+v", err)
			}
			e.T().Logf("%s: %s", message, string(resBytes))

			_ = res.Close()
		}

		e.T().Logf("Pull %q", image)
		res, err := c.ImagePull(ctx, image, types.ImagePullOptions{})
		if err != nil {
			return nil, err
		}
		logResp("pull image", res)

		//listenAddr := sf.FreeLocalTCPAddress(e)
		listenAddr := "localhost:2136"
		host, port, err := net.SplitHostPort(listenAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to split host port: %+v", err)
		}

		containerPort, err := nat.NewPort("tcp", "2136")
		if err != nil {
			e.T().Logf("failed to create nat port: %+v", err)
		}

		hostBinding := nat.PortBinding{
			HostIP:   host,
			HostPort: port,
		}
		portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

		hostConfig := &container.HostConfig{PortBindings: portBinding}

		// create container
		resp, err := c.ContainerCreate(ctx, &container.Config{
			Hostname: "localhost",
			Image:    image,
			Env: []string{
				"GRPC_PORT=" + port,
				"YDB_USE_IN_MEMORY_PDISKS=true",
			},
			Labels: map[string]string{label: "1"},
		}, hostConfig, nil, nil, "")
		if err != nil {
			e.T().Logf("failed to create container: %+v", err)
		}

		e.T().Logf("Created container: %q", resp.ID)

		clean := func() {
			err := c.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{
				Force: true,
			})
			log.Printf("removed container %q: %+v", resp.ID, err)
		}
		defer func() {
			if resErr != nil {
				clean()
			}
		}()

		// start container
		err = c.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
		if err != nil {
			return nil, err
		}

		e.T().Logf("waiting container...")
		var status string
	waitContainerLoop:
		for {
			description, err := c.ContainerInspect(ctx, resp.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to inspect container: %w", err)
			}
			status = strings.ToLower(description.State.Health.Status)
			if status != "starting" {
				break waitContainerLoop
			}
			time.Sleep(time.Second / 10)
		}

		if status != "healthy" {
			return nil, fmt.Errorf("failed to start container, state: %q", status)
		}
		return fixenv.NewGenericResultWithCleanup(listenAddr, clean), nil
	}

	return fixenv.CacheResult(e, f, fixenv.CacheOptions{Scope: fixenv.ScopePackage})
}
