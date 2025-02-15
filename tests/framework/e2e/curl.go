// Copyright 2021 The etcd Authors
//
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

package e2e

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type CURLReq struct {
	Username string
	Password string

	IsTLS   bool
	Timeout int

	Endpoint string

	Value    string
	Expected string
	Header   string

	MetricsURLScheme string

	Ciphers string
}

func (r CURLReq) timeoutDuration() time.Duration {
	if r.Timeout != 0 {
		return time.Duration(r.Timeout) * time.Second
	}

	// assume a sane default to finish a curl request
	return 5 * time.Second
}

// CURLPrefixArgs builds the beginning of a curl command for a given key
// addressed to a random URL in the given cluster.
func CURLPrefixArgs(cfg *EtcdProcessClusterConfig, member EtcdProcess, method string, req CURLReq) []string {
	var (
		cmdArgs = []string{"curl"}
		acurl   = member.Config().Acurl
	)
	if req.MetricsURLScheme != "https" {
		if req.IsTLS {
			if cfg.Client.ConnectionType != ClientTLSAndNonTLS {
				panic("should not use cURLPrefixArgsUseTLS when serving only TLS or non-TLS")
			}
			cmdArgs = append(cmdArgs, "--cacert", CaPath, "--cert", CertPath, "--key", PrivateKeyPath)
			acurl = ToTLS(member.Config().Acurl)
		} else if cfg.Client.ConnectionType == ClientTLS {
			if !cfg.NoCN {
				cmdArgs = append(cmdArgs, "--cacert", CaPath, "--cert", CertPath, "--key", PrivateKeyPath)
			} else {
				cmdArgs = append(cmdArgs, "--cacert", CaPath, "--cert", CertPath3, "--key", PrivateKeyPath3)
			}
		}
	}
	if req.MetricsURLScheme != "" {
		acurl = member.EndpointsMetrics()[0]
	}
	ep := acurl + req.Endpoint

	if req.Username != "" || req.Password != "" {
		cmdArgs = append(cmdArgs, "-L", "-u", fmt.Sprintf("%s:%s", req.Username, req.Password), ep)
	} else {
		cmdArgs = append(cmdArgs, "-L", ep)
	}
	if req.Timeout != 0 {
		cmdArgs = append(cmdArgs, "-m", fmt.Sprintf("%d", req.Timeout))
	}

	if req.Header != "" {
		cmdArgs = append(cmdArgs, "-H", req.Header)
	}

	if req.Ciphers != "" {
		cmdArgs = append(cmdArgs, "--ciphers", req.Ciphers)
	}

	switch method {
	case "POST", "PUT":
		dt := req.Value
		if !strings.HasPrefix(dt, "{") { // for non-JSON value
			dt = "value=" + dt
		}
		cmdArgs = append(cmdArgs, "-X", method, "-d", dt)
	}
	return cmdArgs
}

func CURLPost(clus *EtcdProcessCluster, req CURLReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), req.timeoutDuration())
	defer cancel()
	return SpawnWithExpectsContext(ctx, CURLPrefixArgs(clus.Cfg, clus.Procs[rand.Intn(clus.Cfg.ClusterSize)], "POST", req), nil, req.Expected)
}

func CURLPut(clus *EtcdProcessCluster, req CURLReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), req.timeoutDuration())
	defer cancel()
	return SpawnWithExpectsContext(ctx, CURLPrefixArgs(clus.Cfg, clus.Procs[rand.Intn(clus.Cfg.ClusterSize)], "PUT", req), nil, req.Expected)
}

func CURLGet(clus *EtcdProcessCluster, req CURLReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), req.timeoutDuration())
	defer cancel()

	return SpawnWithExpectsContext(ctx, CURLPrefixArgs(clus.Cfg, clus.Procs[rand.Intn(clus.Cfg.ClusterSize)], "GET", req), nil, req.Expected)
}
