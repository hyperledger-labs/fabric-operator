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

package v1_test

import (
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v1"
	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/pointer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	certB64 = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBdFJBUDlMemUyZEc1cm1rbmcvdVVtREFZU0VwUElqRFdUUDhqUjMxcUJ5Yjc3YWUrCnk3UTRvRnZod1lDVUhsUWVTWjFKeTdUUHpEcitoUk5hdDJYNGdGYUpGYmVFbC9DSHJ3Rk1mNzNzQStWV1pHdnkKdXhtbjB2bEdYMW5zSEo5aUdIUS9qR2FvV1FJYzlVbnpHWi8yWStlZkpxOWd3cDBNemFzWWZkdXordXVBNlp4VAp5TTdDOWFlWmxYL2ZMYmVkSXVXTzVzaXhPSlZQeUVpcWpkd0RiY1AxYy9mRCtSMm1DbmM3VGovSnVLK1poTGxPCnhGcVlFRmtROHBmSi9LY1pabVF1QURZVFh6RGp6OENxcTRTRU5ySzI0b2hQQkN2SGgyanplWjhGdGR4MmpSSFQKaXdCZWZEYWlSWVBSOUM4enk4K1Z2Wmt6S0hQV3N5aENiNUMrN1FJREFRQUJBb0lCQUZROGhzL2IxdW9Mc3BFOApCdEJXaVVsTWh0K0xBc25yWXFncnd5UU5hdmlzNEdRdXVJdFk2MGRmdCtZb2hjQ2ViZ0RkbG1tWlUxdTJ6cGJtCjdEdUt5MVFaN21rV0dpLytEWUlUM3AxSHBMZ2pTRkFzRUorUFRnN1BQamc2UTZrRlZjUCt3Vm4yb0xmWVRkU28KZE5zbEdxSmNMaVQzVHRMNzhlcjFnTTE5RzN6T3J1ZndrSGJSYU1BRmtvZ1ExUlZLSWpnVGUvbmpIMHFHNW9JagoxNEJLeFFKTUZFTG1pQk50NUx5OVMxWWdxTDRjbmNtUDN5L1QyNEdodVhNckx0eTVOeVhnS0dFZ1pUTDMzZzZvCnYreDFFMFRURWRjMVQvWVBGWkdBSXhHdWRKNWZZZ2JtWU9LZ09mUHZFOE9TbEV6OW56aHNnckVZYjdQVThpZDUKTHFycVJRRUNnWUVBNjIyT3RIUmMxaVY1ZXQxdHQydTVTTTlTS2h2b0lPT3d2Q3NnTEI5dDJzNEhRUlRYN0RXcAo0VDNpUC9leEl5OXI3bTIxNFo5MEgzZlpVNElSUkdHSUxKUVMrYzRQNVA4cHJFTDcyd1dIWlpQTTM3QlZTQ1U3CkxOTXl4TkRjeVdjSUJIVFh4NUY2eXhLNVFXWTg5MVB0eDlDamJFSEcrNVJVdDA4UVlMWDlUQTBDZ1lFQXhPSmYKcXFjeThMOVZyYUFVZG9lbGdIU0NGSkJRR3hMRFNSQlJSTkRIOUJhaWlZOCtwZzd2TExTRXFMRFpsbkZPbFkrQQpiRENEQ0RtdHhwRXViY0x6b3FnOXhlQTZ0eXZZWkNWalY5dXVzNVh1Wmk1VDBBUHhCdm56OHNNa3dRY3RQWkRQCk8zQTN4WllkZzJBRmFrV1BmT1FFbjVaK3F4TU13SG9VZ1ZwQkptRUNnWUJ2Q2FjcTJVOEgrWGpJU0ROOU5TT1kKZ1ovaEdIUnRQcmFXcVVodFJ3MkxDMjFFZHM0NExEOUphdVNSQXdQYThuelhZWXROTk9XU0NmYkllaW9tdEZHRApwUHNtTXRnd1MyQ2VUS0Y0OWF5Y2JnOU0yVi8vdlAraDdxS2RUVjAwNkpGUmVNSms3K3FZYU9aVFFDTTFDN0swCmNXVUNwQ3R6Y014Y0FNQmF2THNRNlFLQmdHbXJMYmxEdjUxaXM3TmFKV0Z3Y0MwL1dzbDZvdVBFOERiNG9RV1UKSUowcXdOV2ZvZm95TGNBS3F1QjIrbkU2SXZrMmFiQ25ZTXc3V0w4b0VJa3NodUtYOVgrTVZ6Y1VPekdVdDNyaQpGeU9mcHJJRXowcm5zcWNSNUJJNUZqTGJqVFpyMEMyUWp2NW5FVFAvaHlpQWFRQ1l5THAyWlVtZ0Vjb0VPNWtwClBhcEJBb0dBZVV0WjE0SVp2cVorQnAxR1VqSG9PR0pQVnlJdzhSRUFETjRhZXRJTUlQRWFVaDdjZUtWdVN6VXMKci9WczA1Zjg0cFBVaStuUTUzaGo2ZFhhYTd1UE1aMFBnNFY4cS9UdzJMZ3BWWndVd0ltZUQrcXNsbldha3VWMQpMSnp3SkhOa3pOWE1OMmJWREFZTndSamNRSmhtbzF0V2xHYlpRQjNoSkEwR2thWGZPa2c9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg=="
)

var _ = Describe("Peer configuration", func() {
	Context("reading and writing peer configuration file", func() {
		BeforeEach(func() {
			coreConfig := &config.Core{
				Core: v1.Core{
					Peer: v1.Peer{
						ID: "test",
					},
				},
			}

			err := coreConfig.WriteToFile("/tmp/core.yaml")
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates core.yaml", func() {
			Expect("/tmp/core.yaml").Should(BeAnExistingFile())
		})

		It("read core.yaml", func() {
			core, err := config.ReadCoreFile("/tmp/core.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(core.Peer.ID).To(Equal("test"))
		})
	})

	It("merges current configuration with overrides values", func() {
		core, err := config.ReadCoreFile("../../../../../testdata/init/peer/core.yaml")
		Expect(err).NotTo(HaveOccurred())
		Expect(core.Peer.ID).To(Equal("jdoe"))

		newConfig := &config.Core{
			Core: v1.Core{
				Peer: v1.Peer{
					ID: "test",
					BCCSP: &common.BCCSP{
						ProviderName: "PKCS11",
						PKCS11: &common.PKCS11Opts{
							Library:    "library2",
							Label:      "label2",
							Pin:        "2222",
							HashFamily: "SHA3",
							SecLevel:   512,
							FileKeyStore: &common.FileKeyStoreOpts{
								KeyStorePath: "keystore3",
							},
						},
					},
					Discovery: v1.Discovery{
						Enabled: pointer.False(),
					},
					Keepalive: v1.KeepAlive{
						MinInterval: common.MustParseDuration("13s"),
					},
					DeliveryClient: v1.DeliveryClient{
						AddressOverrides: []v1.AddressOverride{
							v1.AddressOverride{
								From:        "old",
								To:          "new",
								CACertsFile: certB64,
							},
						},
					},
				},
			},
		}

		Expect(core.Peer.Keepalive.MinInterval).To(Equal(common.MustParseDuration("60s")))

		err = core.MergeWith(newConfig, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(core.Peer.ID).To(Equal("test"))
		Expect(core.Peer.BCCSP.PKCS11.Library).To(Equal("/usr/local/lib/libpkcs11-proxy.so"))
		Expect(core.Peer.BCCSP.PKCS11.Label).To(Equal("label2"))
		Expect(core.Peer.BCCSP.PKCS11.Pin).To(Equal("2222"))
		Expect(core.Peer.BCCSP.PKCS11.HashFamily).To(Equal("SHA3"))
		Expect(core.Peer.BCCSP.PKCS11.SecLevel).To(Equal(512))
		Expect(core.Peer.BCCSP.PKCS11.FileKeyStore.KeyStorePath).To(Equal("keystore3"))

		Expect(core.Peer.Keepalive.MinInterval).To(Equal(common.MustParseDuration("13s")))

		Expect(core.Peer.DeliveryClient.AddressOverrides[0].From).To(Equal("old"))
		Expect(core.Peer.DeliveryClient.AddressOverrides[0].To).To(Equal("new"))
		Expect(core.Peer.DeliveryClient.AddressOverrides[0].CACertsFile).To(Equal("/orderer/certs/cert0.pem"))

		Expect(*core.Peer.Discovery.Enabled).To(Equal(false))
	})

	It("merges with default values", func() {
		core, err := config.ReadCoreFile("../../../../../testdata/init/peer/core.yaml")
		Expect(err).NotTo(HaveOccurred())
		Expect(core.Peer.ID).To(Equal("jdoe"))

		newConfig := &config.Core{
			Core: v1.Core{
				Peer: v1.Peer{
					ID: "test",
					BCCSP: &common.BCCSP{
						ProviderName: "PKCS11",
						PKCS11: &common.PKCS11Opts{
							Label: "label2",
							Pin:   "2222",
						},
					},
					Discovery: v1.Discovery{
						Enabled: pointer.False(),
					},
				},
			},
		}

		err = core.MergeWith(newConfig, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(core.Peer.ID).To(Equal("test"))
		Expect(core.Peer.BCCSP.PKCS11.Library).To(Equal("/usr/local/lib/libpkcs11-proxy.so"))
		Expect(core.Peer.BCCSP.PKCS11.Label).To(Equal("label2"))
		Expect(core.Peer.BCCSP.PKCS11.Pin).To(Equal("2222"))
		Expect(core.Peer.BCCSP.PKCS11.HashFamily).To(Equal("SHA2"))
		Expect(core.Peer.BCCSP.PKCS11.SecLevel).To(Equal(256))
		Expect(core.Peer.BCCSP.PKCS11.FileKeyStore.KeyStorePath).To(Equal("keystore2"))
	})

	It("reads in core.yaml and unmarshal it to peer config", func() {
		core, err := config.ReadCoreFile("../../../../../testdata/init/peer/core.yaml")
		Expect(err).NotTo(HaveOccurred())

		peerConfig := core.Peer
		By("setting ID", func() {
			Expect(peerConfig.ID).To(Equal("jdoe"))
		})

		By("setting NetworkID", func() {
			Expect(peerConfig.NetworkID).To(Equal("dev"))
		})

		By("setting ListenAddress", func() {
			Expect(peerConfig.ListenAddress).To(Equal("0.0.0.0:7051"))
		})

		By("setting ChaincodeListenAddress", func() {
			Expect(peerConfig.ChaincodeListenAddress).To(Equal("0.0.0.0:7052"))
		})

		By("setting ChaincodeAddress", func() {
			Expect(peerConfig.ChaincodeAddress).To(Equal("0.0.0.0:7053"))
		})

		By("setting Address", func() {
			Expect(peerConfig.Address).To(Equal("0.0.0.0:7054"))
		})

		By("setting AddressAutoDetect", func() {
			Expect(*peerConfig.AddressAutoDetect).To(Equal(true))
		})

		By("setting FileSystemPath", func() {
			Expect(peerConfig.FileSystemPath).To(Equal("/var/hyperledger/production"))
		})

		By("setting MspConfigPath", func() {
			Expect(peerConfig.MspConfigPath).To(Equal("msp"))
		})

		By("setting LocalMspId", func() {
			Expect(peerConfig.LocalMspId).To(Equal("SampleOrg"))
		})

		By("setting LocalMspType", func() {
			Expect(peerConfig.LocalMspType).To(Equal("bccsp"))
		})

		By("setting ValidatorPoolSize", func() {
			Expect(peerConfig.ValidatorPoolSize).To(Equal(5))
		})
		// KeepAlive

		By("setting Keepalive.MinInterval", func() {
			d, err := common.ParseDuration("60s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Keepalive.MinInterval).To(Equal(d))
		})

		By("setting Keepalive.Client.Interval", func() {
			d, err := common.ParseDuration("60s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Keepalive.Client.Interval).To(Equal(d))
		})

		By("setting Keepalive.Client.Timeout", func() {
			d, err := common.ParseDuration("20s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Keepalive.Client.Timeout).To(Equal(d))
		})

		By("setting Keepalive.DeliveryClient.Interval", func() {
			d, err := common.ParseDuration("60s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Keepalive.DeliveryClient.Interval).To(Equal(d))
		})

		By("setting Keepalive.DeliveryClient.Timeout", func() {
			d, err := common.ParseDuration("20s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Keepalive.DeliveryClient.Timeout).To(Equal(d))
		})

		// Gossip
		By("setting Gossip.Bootstrap", func() {
			Expect(peerConfig.Gossip.Bootstrap).To(Equal([]string{"127.0.0.1:7051", "127.0.0.1:7052"}))
		})

		By("setting Gossip.UseLeaderElection", func() {
			Expect(*peerConfig.Gossip.UseLeaderElection).To(Equal(true))
		})

		By("setting Gossip.OrgLeader", func() {
			Expect(*peerConfig.Gossip.OrgLeader).To(Equal(true))
		})

		By("setting Gossip.MembershipTrackerInterval", func() {
			d, err := common.ParseDuration("5s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.MembershipTrackerInterval).To(Equal(d))
		})

		By("setting Gossip.Endpoint", func() {
			Expect(peerConfig.Gossip.Endpoint).To(Equal("endpoint1"))
		})

		By("setting Gossip.MaxBlockCountToStore", func() {
			Expect(peerConfig.Gossip.MaxBlockCountToStore).To(Equal(10))
		})

		By("setting Gossip.MaxPropogationBurstLatency", func() {
			d, err := common.ParseDuration("10ms")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.MaxPropagationBurstLatency).To(Equal(d))
		})

		By("setting Gossip.MaxPropogationBurstSize", func() {
			Expect(peerConfig.Gossip.MaxPropagationBurstSize).To(Equal(10))
		})

		By("setting Gossip.PropagateIterations", func() {
			Expect(peerConfig.Gossip.PropagateIterations).To(Equal(1))
		})

		By("setting Gossip.PropagatePeerNum", func() {
			Expect(peerConfig.Gossip.PropagatePeerNum).To(Equal(3))
		})

		By("setting Gossip.PullInterval", func() {
			d, err := common.ParseDuration("4s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.PullInterval).To(Equal(d))
		})

		By("setting Gossip.PullPeerNum", func() {
			Expect(peerConfig.Gossip.PullPeerNum).To(Equal(3))
		})

		By("setting Gossip.RequestStateInfoInterval", func() {
			d, err := common.ParseDuration("4s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.RequestStateInfoInterval).To(Equal(d))
		})

		By("setting Gossip.PublishStateInfoInterval", func() {
			d, err := common.ParseDuration("4s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.PublishStateInfoInterval).To(Equal(d))
		})

		By("setting Gossip.StateInfoRetentionInterval", func() {
			d, err := common.ParseDuration("2s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.StateInfoRetentionInterval).To(Equal(d))
		})

		By("setting Gossip.PublishCertPeriod", func() {
			d, err := common.ParseDuration("10s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.PublishCertPeriod).To(Equal(d))
		})

		By("setting Gossip.SkipBlockVerification", func() {
			Expect(*peerConfig.Gossip.SkipBlockVerification).To(Equal(true))
		})

		By("setting Gossip.DialTimeout", func() {
			d, err := common.ParseDuration("3s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.DialTimeout).To(Equal(d))
		})

		By("setting Gossip.ConnTimeout", func() {
			d, err := common.ParseDuration("2s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.ConnTimeout).To(Equal(d))
		})

		By("setting Gossip.RecvBuffSize", func() {
			Expect(peerConfig.Gossip.RecvBuffSize).To(Equal(20))
		})

		By("setting Gossip.SendBuffSize", func() {
			Expect(peerConfig.Gossip.SendBuffSize).To(Equal(200))
		})

		By("setting Gossip.DigestWaitTime", func() {
			d, err := common.ParseDuration("1s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.DigestWaitTime).To(Equal(d))
		})

		By("setting Gossip.RequestWaitTime", func() {
			d, err := common.ParseDuration("1500ms")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.RequestWaitTime).To(Equal(d))
		})

		By("setting Gossip.ResponseWaitTime", func() {
			d, err := common.ParseDuration("2s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.ResponseWaitTime).To(Equal(d))
		})

		By("setting Gossip.AliveTimeInterval", func() {
			d, err := common.ParseDuration("5s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.AliveTimeInterval).To(Equal(d))
		})

		By("setting Gossip.AliveExpirationTimeout", func() {
			d, err := common.ParseDuration("25s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.AliveExpirationTimeout).To(Equal(d))
		})

		By("setting Gossip.ReconnectInterval", func() {
			d, err := common.ParseDuration("25s")
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Gossip.ReconnectInterval).To(Equal(d))
		})

		By("setting Gossip.ExternalEndpoint", func() {
			Expect(peerConfig.Gossip.ExternalEndpoint).To(Equal("externalEndpoint1"))
		})

		// BCCSP
		By("setting BCCSP.ProviderName", func() {
			Expect(peerConfig.BCCSP.ProviderName).To(Equal("SW"))
		})

		By("setting BCCSP.SW.HashFamily", func() {
			Expect(peerConfig.BCCSP.SW.HashFamily).To(Equal("SHA2"))
		})

		By("setting BCCSP.SW.SecLevel", func() {
			Expect(peerConfig.BCCSP.SW.SecLevel).To(Equal(256))
		})

		By("setting BCCSP.SW.FileKeystore.KeystorePath", func() {
			Expect(peerConfig.BCCSP.SW.FileKeyStore.KeyStorePath).To(Equal("keystore1"))
		})

		By("setting BCCSP.PKCS11.Library", func() {
			Expect(peerConfig.BCCSP.PKCS11.Library).To(Equal("library1"))
		})

		By("setting BCCSP.PKCS11.Label", func() {
			Expect(peerConfig.BCCSP.PKCS11.Label).To(Equal("label1"))
		})

		By("setting BCCSP.PKCS11.Pin", func() {
			Expect(peerConfig.BCCSP.PKCS11.Pin).To(Equal("1234"))
		})

		By("setting BCCSP.PKCS11.HashFamily", func() {
			Expect(peerConfig.BCCSP.PKCS11.HashFamily).To(Equal("SHA2"))
		})

		By("setting BCCSP.PKCS11.Security", func() {
			Expect(peerConfig.BCCSP.PKCS11.SecLevel).To(Equal(256))
		})

		By("setting BCCSP.PKCS11.FileKeystore.KeystorePath", func() {
			Expect(peerConfig.BCCSP.PKCS11.FileKeyStore.KeyStorePath).To(Equal("keystore2"))
		})

		// Discovery
		By("setting Discovery.Enabled", func() {
			Expect(*peerConfig.Discovery.Enabled).To(Equal(true))
		})

		By("setting Discovery.AuthCacheEnabled", func() {
			Expect(*peerConfig.Discovery.AuthCacheEnabled).To(Equal(true))
		})

		By("setting Discovery.AuthCacheMaxSize", func() {
			Expect(peerConfig.Discovery.AuthCacheMaxSize).To(Equal(1000))
		})

		By("setting Discovery.AuthCachePurgeRetentionRatio", func() {
			Expect(peerConfig.Discovery.AuthCachePurgeRetentionRatio).To(Equal(0.75))
		})

		By("setting Discovery.OrgMembersAllowedAccess", func() {
			Expect(*peerConfig.Discovery.OrgMembersAllowedAccess).To(Equal(true))
		})

		By("setting Limits.Concurrency.Qscc", func() {
			Expect(peerConfig.Limits.Concurrency.Qscc).To(Equal(5000))
		})

		// Handlers
		By("setting Handlers.AuthFilters", func() {
			Expect(peerConfig.Handlers.AuthFilters).To(Equal([]v1.HandlerConfig{
				v1.HandlerConfig{
					Name: "DefaultAuth",
				},
				v1.HandlerConfig{
					Name: "ExpirationCheck",
				},
			}))
		})

		By("setting Handlers.Decorators", func() {
			Expect(peerConfig.Handlers.Decorators).To(Equal([]v1.HandlerConfig{
				v1.HandlerConfig{
					Name: "DefaultDecorator",
				},
			}))
		})

		By("setting Handlers.Endorsers", func() {
			Expect(peerConfig.Handlers.Endorsers).To(Equal(v1.PluginMapping{
				"escc": v1.HandlerConfig{
					Name: "DefaultEndorsement",
				},
			}))
		})

		By("setting Handlers.Validators", func() {
			Expect(peerConfig.Handlers.Validators).To(Equal(v1.PluginMapping{
				"vscc": v1.HandlerConfig{
					Name: "DefaultValidation",
				},
			}))
		})
	})

	Context("chaincode configuration", func() {
		It("reads in core.yaml and unmarshal it to chaincode config", func() {
			core, err := config.ReadCoreFile("../../../../../testdata/init/peer/core.yaml")
			Expect(err).NotTo(HaveOccurred())

			chaincode := core.Chaincode
			By("setting Chaincode.StartupTimeout", func() {
				d, err := common.ParseDuration("300s")
				Expect(err).NotTo(HaveOccurred())
				Expect(chaincode.StartupTimeout).To(Equal(d))
			})

			By("setting Chaincode.ExecuteTimeout", func() {
				d, err := common.ParseDuration("30s")
				Expect(err).NotTo(HaveOccurred())
				Expect(chaincode.ExecuteTimeout).To(Equal(d))
			})
		})

		It("merges current configuration with overrides values", func() {
			core, err := config.ReadCoreFile("../../../../../testdata/init/peer/core.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(core.Peer.ID).To(Equal("jdoe"))

			startupTimeout, err := common.ParseDuration("200s")
			Expect(err).NotTo(HaveOccurred())
			executeTimeout, err := common.ParseDuration("20s")
			Expect(err).NotTo(HaveOccurred())

			newConfig := &config.Core{
				Core: v1.Core{
					Chaincode: v1.Chaincode{
						StartupTimeout: startupTimeout,
						ExecuteTimeout: executeTimeout,
					},
				},
			}

			err = core.MergeWith(newConfig, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(core.Chaincode.StartupTimeout).To(Equal(startupTimeout))
			Expect(core.Chaincode.ExecuteTimeout).To(Equal(executeTimeout))
		})
	})

	Context("DeliveryClient.AddressOverrides", func() {
		It("merges current configuration with overrides values", func() {
			core, err := config.ReadCoreFile("../../../../../testdata/init/peer/core.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(core.Peer.ID).To(Equal("jdoe"))

			addressOverrides := []v1.AddressOverride{
				v1.AddressOverride{
					From:        "address_old",
					To:          "address_new",
					CACertsFile: certB64,
				},
			}

			newConfig := &config.Core{
				Core: v1.Core{
					Peer: v1.Peer{
						DeliveryClient: v1.DeliveryClient{
							AddressOverrides: addressOverrides,
						},
					},
				},
			}

			err = core.MergeWith(newConfig, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(core.Peer.DeliveryClient.AddressOverrides[0].From).To(Equal(addressOverrides[0].From))
			Expect(core.Peer.DeliveryClient.AddressOverrides[0].To).To(Equal(addressOverrides[0].To))
			Expect(core.Peer.DeliveryClient.AddressOverrides[0].CACertsFile).To(Equal("/orderer/certs/cert0.pem"))
			Expect(len(core.GetAddressOverrides()[0].GetCertBytes())).NotTo(Equal(0))
		})
	})

	Context("operations configuration", func() {
		It("merges current configuration with overrides values", func() {
			core, err := config.ReadCoreFile("../../../../../testdata/init/peer/core.yaml")
			Expect(err).NotTo(HaveOccurred())

			Expect(core.Operations.ListenAddress).To(Equal("127.0.0.1:9443"))
			Expect(*core.Operations.TLS.Enabled).To(Equal(false))
			Expect(core.Operations.TLS.Certificate.File).To(Equal("cert.pem"))
			Expect(core.Operations.TLS.PrivateKey.File).To(Equal("key.pem"))
			Expect(*core.Operations.TLS.ClientAuthRequired).To(Equal(false))
			Expect(core.Operations.TLS.ClientRootCAs.Files).To(Equal([]string{"rootcert.pem"}))

			newConfig := &config.Core{
				Core: v1.Core{
					Operations: v1.Operations{
						ListenAddress: "localhost:8080",
						TLS: v1.OperationsTLS{
							Enabled: pointer.True(),
							Certificate: v1.File{
								File: "newcert.pem",
							},
							PrivateKey: v1.File{
								File: "newkey.pem",
							},
							ClientAuthRequired: pointer.True(),
							ClientRootCAs: v1.Files{
								Files: []string{"newrootcert.pem", "newrootcert2.pem"},
							},
						},
					},
				},
			}

			err = core.MergeWith(newConfig, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(core.Operations.ListenAddress).To(Equal("localhost:8080"))
			Expect(*core.Operations.TLS.Enabled).To(Equal(true))
			Expect(core.Operations.TLS.Certificate.File).To(Equal("newcert.pem"))
			Expect(core.Operations.TLS.PrivateKey.File).To(Equal("newkey.pem"))
			Expect(*core.Operations.TLS.ClientAuthRequired).To(Equal(true))
			Expect(core.Operations.TLS.ClientRootCAs.Files).To(Equal([]string{"newrootcert.pem", "newrootcert2.pem"}))
		})
	})

	Context("metrics configuration", func() {
		It("merges current configuration with overrides values", func() {
			core, err := config.ReadCoreFile("../../../../../testdata/init/peer/core.yaml")
			Expect(err).NotTo(HaveOccurred())

			Expect(core.Metrics.Provider).To(Equal("prometheus"))
			Expect(core.Metrics.Statsd.Network).To(Equal("udp"))
			Expect(core.Metrics.Statsd.Address).To(Equal("127.0.0.1:8125"))
			Expect(core.Metrics.Statsd.Prefix).To(Equal(""))

			writeInterval, err := common.ParseDuration("10s")
			Expect(err).NotTo(HaveOccurred())
			Expect(core.Metrics.Statsd.WriteInterval).To(Equal(writeInterval))

			newWriteInterval, err := common.ParseDuration("15s")
			Expect(err).NotTo(HaveOccurred())
			newConfig := &config.Core{
				Core: v1.Core{
					Metrics: v1.Metrics{
						Provider: "statsd",
						Statsd: v1.Statsd{
							Network:       "tcp",
							Address:       "localhost:8080",
							WriteInterval: newWriteInterval,
							Prefix:        "prefix",
						},
					},
				},
			}

			err = core.MergeWith(newConfig, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(core.Metrics.Provider).To(Equal("statsd"))
			Expect(core.Metrics.Statsd.Network).To(Equal("tcp"))
			Expect(core.Metrics.Statsd.Address).To(Equal("localhost:8080"))
			Expect(core.Metrics.Statsd.Prefix).To(Equal("prefix"))
			Expect(core.Metrics.Statsd.WriteInterval).To(Equal(newWriteInterval))
		})
	})

	Context("updating peer.gossip.bootstrap if needed", func() {
		It("reads core and converts peer.gossip.bootstrap", func() {
			core, err := config.ReadCoreFile("../../../../../testdata/init/peer/core_bootstrap_test.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(core.Peer.Gossip.Bootstrap).To(Equal([]string{"127.0.0.1:7051"}))
		})

		It("returns error if invalid core (besides peer.gossip.boostrap field)", func() {
			_, err := config.ReadCoreFile("../../../../../testdata/init/peer/core_invalid.yaml")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
