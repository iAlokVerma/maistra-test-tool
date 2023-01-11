// Copyright 2021 Red Hat, Inc.
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

package egress

import (
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupEgressTLSOrigination() {
	util.Log.Info("Cleanup")
	sleep := examples.Sleep{"bookinfo"}
	util.KubeDeleteContents("bookinfo", ExServiceEntryOriginate)
	util.KubeDeleteContents("bookinfo", ExServiceEntry)
	util.KubeDeleteContents("bookinfo", CiscoProxy)
	sleep.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestEgressTLSOrigination(t *testing.T) {
	defer cleanupEgressTLSOrigination()
	defer util.RecoverPanic(t)

	util.Log.Info("TestEgressTLSOrigination")
	sleep := examples.Sleep{"bookinfo"}
	sleep.Install()
	sleepPod, err := util.GetPodName("bookinfo", "app=sleep")
	util.Inspect(err, "Failed to get sleep pod name", "", t)
	util.BashShell(`kubectl cp /home/noiro/http_server/ca-cert.pem %s:/tmp/ -c sleep`,sleepPod)
	//command := `curl http://istio.io/ca-cert.pem >> /tmp/ca-cert.pem`
	//msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)

	t.Run("TrafficManagement_egress_configure_access_to_external_service", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Create a ServiceEntry for cisco proxy")
		util.KubeApplyContents("bookinfo", CiscoProxy)
		time.Sleep(time.Duration(10) * time.Second)

		util.Log.Info("Create a ServiceEntry to external istio.io")
		util.KubeApplyContents("bookinfo", ExServiceEntry)
		time.Sleep(time.Duration(10) * time.Second)

		//command := `curl --proxy http://proxy.esl.cisco.com:80 -sSL -o /dev/null -D - http://istio.io`
		command := `curl -sSL --cacert /tmp/ca-cert.pem -o /dev/null -D - http://istio.io`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") {
			util.Log.Info("Success. Get http://istio.io response")
		} else {
			util.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}
	})

	t.Run("TrafficManagement_egress_tls_origination", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("TLS origination for egress traffic")
		util.KubeApplyContents("bookinfo", ExServiceEntryOriginate)
		time.Sleep(time.Duration(10) * time.Second)

		//command := `curl --proxy http://proxy.esl.cisco.com:80 -sSL -o /dev/null -D - http://istio.io`
		command := `curl -sSL -o /dev/null -D - http://istio.io`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") || strings.Contains(msg, "503 Service Unavailable") {
			util.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		} else {
			util.Log.Infof("Success. Get http://istio.io response: %s", msg)
		}

		//command = `curl --proxy http://proxy.esl.cisco.com:80 -sSL -o /dev/null -D - https://istio.io`
		command = `curl -k --cacert /tmp/ca-cert.pem -sSL -o /dev/null -D - https://istio.io`
		msg, err = util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301 Moved Permanently") || strings.Contains(msg, "503 Service Unavailable") {
			util.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		} else {
			util.Log.Infof("Success. Get https://istio.io response: %s", msg)
		}
	})
}
