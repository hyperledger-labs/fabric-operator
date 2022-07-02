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

package restart_test

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart/staggerrestarts"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Restart", func() {
	SetDefaultEventuallyTimeout(30 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	var (
		mockClient *controllermocks.Client
		instance   *current.IBPPeer

		restartManager *restart.RestartManager

		cfg           *restart.Config
		updatedCfg    *restart.Config
		testTimestamp string
	)

	BeforeEach(func() {
		mockClient = &controllermocks.Client{}
		restartManager = restart.New(mockClient, 10*time.Minute, 5*time.Minute)

		instance = &current.IBPPeer{}
		instance.Name = "peer1"
		instance.Namespace = "default"

		testTimestamp = time.Now().UTC().Format(time.RFC3339)
		cfg = &restart.Config{
			Instances: map[string]*restart.Restart{
				"peer1": {
					Requests: map[restart.Reason]*restart.Request{
						restart.ADMINCERT: {
							RequestTimestamp: testTimestamp,
							Status:           restart.Pending,
						},
					},
				},
				"peer2": {
					Requests: map[restart.Reason]*restart.Request{
						restart.ADMINCERT: {
							LastActionTimestamp: time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
						},
					},
				},
				"peer3": {
					Requests: map[restart.Reason]*restart.Request{
						restart.ADMINCERT: {
							LastActionTimestamp: time.Now().Add(-5 * time.Second).UTC().Format(time.RFC3339),
							RequestTimestamp:    testTimestamp,
							Status:              restart.Pending,
						},
					},
				},
				"peer4": {
					Requests: map[restart.Reason]*restart.Request{
						restart.ADMINCERT: {
							LastActionTimestamp: time.Now().Add(-15 * time.Minute).UTC().Format(time.RFC3339),
							RequestTimestamp:    testTimestamp,
							Status:              restart.Pending,
						},
						restart.ECERTUPDATE: {
							LastActionTimestamp: time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
							RequestTimestamp:    testTimestamp,
							Status:              restart.Pending,
						},
					},
				},
			},
		}

		cfgBytes, err := json.Marshal(cfg)
		Expect(err).NotTo(HaveOccurred())

		mockClient.GetStub = func(ctx context.Context, ns types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *corev1.ConfigMap:
				o := obj.(*corev1.ConfigMap)
				switch ns.Name {
				case "operator-config":
					o.Name = "operator-config"
					o.Namespace = instance.Namespace
					o.BinaryData = map[string][]byte{
						"restart-config.yaml": cfgBytes,
					}
				}
			case *appsv1.Deployment:
				o := obj.(*appsv1.Deployment)
				o.Name = ns.Name
				o.Namespace = instance.Namespace
			}

			return nil
		}

		updatedCfg = &restart.Config{}
		mockClient.CreateOrUpdateStub = func(ctx context.Context, obj client.Object, opts ...k8sclient.CreateOrUpdateOption) error {
			o := obj.(*corev1.ConfigMap)
			err := json.Unmarshal(o.BinaryData["restart-config.yaml"], updatedCfg)
			Expect(err).NotTo(HaveOccurred())
			return nil
		}
	})

	Context("get config from config map", func() {
		It("returns empty config if config map doesn't exist", func() {
			mockClient.GetReturns(k8serrors.NewNotFound(schema.GroupResource{}, "not found"))
			config, err := restartManager.GetConfig(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).To(Equal(&restart.Config{}))
		})

		It("returns error if failed to get existing config map", func() {
			mockClient.GetReturns(errors.New("get error"))
			_, err := restartManager.GetConfig(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("failed to get operator-config config map: get error"))
		})

		It("returns error if fails to unmarshal config map", func() {
			mockClient.GetStub = func(ctx context.Context, ns types.NamespacedName, obj client.Object) error {
				o := obj.(*corev1.ConfigMap)
				o.Name = "operator-config"
				o.Namespace = instance.Namespace
				o.BinaryData = map[string][]byte{
					"restart-config.yaml": []byte("invalid"),
				}
				return nil
			}
			_, err := restartManager.GetConfig(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to unmarshal operator-config config map"))
		})

		It("returns restart config from config map", func() {
			config, err := restartManager.GetConfig(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).To(Equal(cfg))
		})
	})

	Context("update config map", func() {
		It("returns error if fails to update config map", func() {
			mockClient.CreateOrUpdateReturns(errors.New("update error"))
			err := restartManager.UpdateConfigMap(cfg, instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to create or update operator-config config map: update error"))
		})

		It("updates config map", func() {
			err := restartManager.UpdateConfigMap(cfg, instance)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("for admin cert update", func() {
		It("returns error if fails to get config from config map", func() {
			mockClient.GetReturns(errors.New("get error"))
			err := restartManager.ForAdminCertUpdate(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to get operator-config config map: get error"))
		})

		It("returns error if fails to update config map", func() {
			mockClient.CreateOrUpdateReturns(errors.New("update error"))
			err := restartManager.ForAdminCertUpdate(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to create or update operator-config config map: update error"))
		})

		It("doesn't set RequestTimestamp if already set", func() {
			instance.Name = "peer1"
			err := restartManager.ForAdminCertUpdate(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedCfg.Instances["peer1"].Requests[restart.ADMINCERT].RequestTimestamp).To(Equal(testTimestamp))
		})

		It("sets RequestTimestamp if not set for that instance", func() {
			instance.Name = "peer2"
			err := restartManager.ForAdminCertUpdate(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedCfg.Instances["peer2"].Requests[restart.ADMINCERT].RequestTimestamp).NotTo(Equal(""))
		})

		It("sets RequestTimestamp for instance if instance not yet in config", func() {
			instance.Name = "newpeer"
			err := restartManager.ForAdminCertUpdate(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedCfg.Instances["newpeer"].Requests[restart.ADMINCERT].RequestTimestamp).NotTo(Equal(""))
		})

	})

	Context("for ecert reenroll", func() {
		It("sets RequestTimestamp for instance if not set for that instance", func() {
			err := restartManager.ForEcertReenroll(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedCfg.Instances["peer1"].Requests[restart.ECERTUPDATE].RequestTimestamp).NotTo(Equal(""))
		})
	})

	Context("for tls reenroll", func() {
		It("sets RequestTimestamp for instance if not set for that instance", func() {
			err := restartManager.ForTLSReenroll(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedCfg.Instances["peer1"].Requests[restart.TLSUPDATE].RequestTimestamp).NotTo(Equal(""))
		})
	})

	Context("for config override", func() {
		It("sets RequestTimestamp for instance if not set for that instance", func() {
			err := restartManager.ForConfigOverride(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedCfg.Instances["peer1"].Requests[restart.CONFIGOVERRIDE].RequestTimestamp).NotTo(Equal(""))
		})
	})

	Context("for migration", func() {
		It("sets RequestTimestamp for instance if not set for that instance", func() {
			err := restartManager.ForMigration(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedCfg.Instances["peer1"].Requests[restart.MIGRATION].RequestTimestamp).NotTo(Equal(""))
		})
	})

	Context("trigger if needed", func() {
		It("returns error if fails to get config map", func() {
			mockClient.GetReturns(errors.New("get error"))
			err := restartManager.TriggerIfNeeded(instance)
			Expect(err).To(HaveOccurred())
		})

		It("returns nil if instance is not in config map", func() {
			instance.Name = "fake peer"
			err := restartManager.TriggerIfNeeded(instance)
			Expect(err).NotTo(HaveOccurred())
		})

		It("triggers restart if there are pending restarts and no previous restart", func() {
			instance.Name = "peer1"
			err := restartManager.TriggerIfNeeded(instance)
			Expect(err).NotTo(HaveOccurred())

			By("clearing restart", func() {
				for _, req := range updatedCfg.Instances["peer1"].Requests {
					Expect(req.Status).To(Equal(restart.Complete))
					Expect(req.RequestTimestamp).To(Equal(""))
					Expect(req.LastActionTimestamp).NotTo(Equal(""))
				}
			})

			By("adding restart request to queue", func() {
				_, cm, _ := mockClient.CreateOrUpdateArgsForCall(1)
				cfgBytes := cm.(*corev1.ConfigMap).BinaryData["restart-config.yaml"]
				restartcfg := &staggerrestarts.RestartConfig{}
				err = json.Unmarshal(cfgBytes, restartcfg)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(restartcfg.Queues[instance.GetMSPID()])).To(Equal(1))
				Expect(restartcfg.Queues[instance.GetMSPID()][0].CRName).To(Equal(instance.Name))
				Expect(restartcfg.Queues[instance.GetMSPID()][0].Reason).To(Equal("adminCert"))
				Expect(restartcfg.Queues[instance.GetMSPID()][0].Status).To(Equal(staggerrestarts.Pending))
			})
		})

		It("returns nil if there are no pending restarts for instance", func() {
			instance.Name = "peer2"
			err := restartManager.TriggerIfNeeded(instance)
			Expect(err).NotTo(HaveOccurred())
		})

		It("sets timer if there are pending restarts but last restart action timestamp is sooner than 10 min", func() {
			instance.Name = "peer3"
			err := restartManager.TriggerIfNeeded(instance)
			Expect(err).NotTo(HaveOccurred())

			By("not updating config map", func() {
				Expect(mockClient.CreateOrUpdateCallCount()).To(Equal(0))
			})

			By("setting timer", func() {
				// timer.Stop() == true means that it was set
				Expect(restartManager.Timers["peer3"].Stop()).To(Equal(true))
			})
		})

		It("triggers restart if there are pending restarts and at least one request last action timestamp is more than 10 min ago", func() {
			instance.Name = "peer4"
			err := restartManager.TriggerIfNeeded(instance)
			Expect(err).NotTo(HaveOccurred())

			By("clearing restart", func() {
				for reason, req := range updatedCfg.Instances["peer4"].Requests {
					Expect(req.Status).To(Equal(restart.Complete))
					Expect(req.RequestTimestamp).To(Equal(""))
					Expect(req.LastActionTimestamp).NotTo(Equal(cfg.Instances["peer4"].Requests[reason].LastActionTimestamp))
				}
			})

			By("adding restart request to queue", func() {
				_, cm, _ := mockClient.CreateOrUpdateArgsForCall(1)
				cfgBytes := cm.(*corev1.ConfigMap).BinaryData["restart-config.yaml"]
				restartcfg := &staggerrestarts.RestartConfig{}
				err = json.Unmarshal(cfgBytes, restartcfg)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(restartcfg.Queues[instance.GetMSPID()])).To(Equal(1))
				Expect(restartcfg.Queues[instance.GetMSPID()][0].CRName).To(Equal(instance.Name))
				Expect(restartcfg.Queues[instance.GetMSPID()][0].Reason).To(ContainSubstring("adminCert"))
				Expect(restartcfg.Queues[instance.GetMSPID()][0].Reason).To(ContainSubstring("ecertUpdate"))
				Expect(restartcfg.Queues[instance.GetMSPID()][0].Status).To(Equal(staggerrestarts.Pending))
			})
		})
	})

	Context("set timer", func() {
		BeforeEach(func() {
			restartManager.WaitTime = 10 * time.Second
		})

		It("returns error if fails to get config map", func() {
			mockClient.GetReturns(errors.New("get error"))
			err := restartManager.SetTimer(instance, "")
			Expect(err).To(HaveOccurred())
		})

		It("sets timer for instance if there are pending restarts", func() {
			instance.Name = "peer3"
			err := restartManager.SetTimer(instance, "")
			Expect(err).NotTo(HaveOccurred())

			// Timer should go off in 5 seconds
			time.Sleep(10 * time.Second)

			By("restarting deployment after timer goes off", func() {
				Expect(restartManager.Timers["peer3"]).To(BeNil())
			})
		})
	})

})
