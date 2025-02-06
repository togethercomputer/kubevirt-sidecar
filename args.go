/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"net"
	"os"
	"path/filepath"
	"strings"

	domainSchema "github.com/togethercomputer/kubevirt-sidecar/pkg/schema"

	"google.golang.org/grpc"
	vmSchema "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
)

const (
	qemuArgsAnnotation           = "qemuargs.vm.kubevirt.io/args"
	onDefineDomainLoggingMessage = "qemuargs hook called"
	qemuv1NS                     = "http://libvirt.org/schemas/domain/qemu/1.0"
)

type infoServer struct {
	Version string
}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "args",
		Versions: []string{
			s.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type v1alpha1Server struct{}
type v1alpha2Server struct{}

func (s v1alpha2Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)
	newDomainXML, err := onDefineDomain(params.GetVmi(), params.GetDomainXML())
	if err != nil {
		return nil, err
	}
	return &hooksV1alpha2.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}
func (s v1alpha2Server) PreCloudInitIso(_ context.Context, params *hooksV1alpha2.PreCloudInitIsoParams) (*hooksV1alpha2.PreCloudInitIsoResult, error) {
	return &hooksV1alpha2.PreCloudInitIsoResult{
		CloudInitData: params.GetCloudInitData(),
	}, nil
}

func (s v1alpha1Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)
	newDomainXML, err := onDefineDomain(params.GetVmi(), params.GetDomainXML())
	if err != nil {
		return nil, err
	}
	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func onDefineDomain(vmiJSON []byte, domainXML []byte) ([]byte, error) {
	log.Log.Info(onDefineDomainLoggingMessage)

	// Print domain xml as base64
	log.Log.Infof("(pre) domain xml: %v", string(base64.StdEncoding.EncodeToString(domainXML)))

	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	// Lets grab the filesystems and log them
	fs := vmiSpec.Spec.Domain.Devices.Filesystems
	for _, fs := range fs {
		log.Log.Infof("filesystem: %v", fs.Name)
	}

	annotations := vmiSpec.GetAnnotations()

	if _, found := annotations[qemuArgsAnnotation]; !found {
		log.Log.Info("qemu args sidecar was requested, but no attributes provided. Doing nothing.")
		return domainXML, nil
	}

	log.Log.Infof("domain xml: %v", string(domainXML))
	domainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(domainXML, &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		panic(err)
	}

	if domainSpec.QEMUCmd == nil {
		domainSpec.QEMUCmd = &domainSchema.Commandline{}
	}

	inputs := strings.Fields(annotations[qemuArgsAnnotation])

	for _, v := range inputs {
		domainSpec.QEMUCmd.QEMUArg = append(domainSpec.QEMUCmd.QEMUArg, domainSchema.Arg{Value: v})
	}

	if len(domainSpec.QEMUCmd.QEMUArg) > 0 {
		domainSpec.XmlNS = qemuv1NS
	}

	fsDevices := domainSpec.Devices.Filesystems
	for _, fsDevice := range fsDevices {
		if fsDevice.Driver.Type == "virtiofs" {
			targetDir := fsDevice.Target.Dir
			log.Log.Infof("Adding cache-size=2G to virtiofs filesystem with tag: %v", targetDir)
			fsDevice.Driver.Queue = "1024"
			fsDevice.Driver.CacheSize = "2G"
		}
	}

	domainSpec.Devices.Filesystems = fsDevices

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %+v", domainSpec)
		panic(err)
	}

	// Print domain xml as base64
	log.Log.Infof("(post) domain xml: %v", string(base64.StdEncoding.EncodeToString(newDomainXML)))

	log.Log.Info("Successfully updated original domain spec with requested disk attributes")

	return newDomainXML, nil
}

func main() {
	log.InitializeLogging("qemu-args-hook-sidecar")

	socketPath := filepath.Join(hooks.HookSocketsSharedDirectory, "args.sock")
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, infoServer{Version: "v1alpha2"})
	hooksV1alpha1.RegisterCallbacksServer(server, v1alpha1Server{})
	hooksV1alpha2.RegisterCallbacksServer(server, v1alpha2Server{})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha2' services on socket %s", socketPath)
	server.Serve(socket)
}
