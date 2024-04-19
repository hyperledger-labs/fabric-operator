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

package command

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/go-logr/zapr"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-lib/leader"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apis "github.com/IBM-Blockchain/fabric-operator/api"
	ibpv1beta1 "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	controller "github.com/IBM-Blockchain/fabric-operator/controllers"
	"github.com/IBM-Blockchain/fabric-operator/defaultconfig/console"
	oconfig "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/migrator"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	openshiftv1 "github.com/openshift/api/config/v1"

	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uberzap "go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var log = logf.Log.WithName("cmd_operator")

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ibpv1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func Operator(operatorCfg *oconfig.Config) error {
	signalHandler := signals.SetupSignalHandler()

	// In local mode, the operator may be launched and debugged directly as a native process without
	// being deployed to a Kubernetes cluster.
	local := os.Getenv("OPERATOR_LOCAL_MODE") == "true"

	return OperatorWithSignal(operatorCfg, signalHandler, true, local)
}

func OperatorWithSignal(operatorCfg *oconfig.Config, signalHandler context.Context, blocking, local bool) error {
	var err error

	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	// pflag.CommandLine.AddFlagSet(flagset)

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	// pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	// pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	if operatorCfg.Logger != nil {

		config := uberzap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		logger, err := config.Build()
		if err != nil {
			return err
		}

		// Wrap the zap.Logger with go-logr/zapr to satisfy the logr.Logger interface
		log := zapr.NewLogger(logger)

		logf.SetLogger(log)
		ctrl.SetLogger(log)
	} else {
		// Use the unstructured log formatter when running locally.
		logf.SetLogger(zap.New(zap.UseDevMode(local)))
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	printVersion()

	watchNamespace := os.Getenv("WATCH_NAMESPACE")
	var operatorNamespace string
	if watchNamespace == "" {
		// Operator is running in all namespace mode
		log.Info("Installing operator in all namespace mode")
		operatorNamespace, err = GetOperatorNamespace()
		if err != nil {
			log.Error(err, "Failed to get operator namespace")
			time.Sleep(15 * time.Second)
			return err
		}
	} else {
		log.Info("Installing operator in own namespace mode", "WATCH_NAMESPACE", watchNamespace)
		operatorNamespace = watchNamespace
	}

	if !local {
		label := os.Getenv("OPERATOR_LABEL_PREFIX")
		if label == "" {
			label = "fabric"
		}
		err = leader.Become(context.TODO(), label+"-operator-lock")
		if err != nil {
			log.Error(err, "Failed to retry for leader lock")
			os.Exit(1)
		}
	} else {
		log.Info("local run detected, skipping leader election")
	}

	var metricsAddr string
	var enableLeaderElection bool

	if flag.Lookup("metrics-addr") == nil {
		flag.StringVar(&metricsAddr, "metrics-addr", ":8383", "The address the metric endpoint binds to.")
	}
	if flag.Lookup("enable-leader-election") == nil {
		flag.BoolVar(&enableLeaderElection, "enable-leader-election", true,
			"Enable leader election for controller manager. "+
				"Enabling this will ensure there is only one active controller manager.")
	}
	flag.Parse()
	config := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		// LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "c30dd930.ibp.com",
		LeaderElectionNamespace: operatorNamespace,
		Namespace:               watchNamespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	log.Info("Registering Components.")

	//This Method Checks if Console deployment Tag in Console Deployment is same as the console tag in the operator
	// binary (if it is not same, it delete the configmaps $consoleObject-deployer and $consoleObject-console)
	CheckForFixPacks(config, operatorNamespace)

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		return err
	}

	//Add route scheme
	if err := routev1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		return err
	}

	//Add clusterversion scheme
	if err := openshiftv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		return err
	}

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ibpv1beta1.AddToScheme(scheme))

	go func() {
		runtime.Gosched()
		mgrSyncContext, mgrSyncContextCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer mgrSyncContextCancel()

		log.Info("Waiting for cache sync")
		if synced := mgr.GetCache().WaitForCacheSync(mgrSyncContext); !synced {
			log.Error(nil, "Timed out waiting for cache sync")
			os.Exit(1)
		}

		log.Info("Cache sync done")

		// Migrate first
		m := migrator.New(mgr, operatorCfg, operatorNamespace)
		err = m.Migrate()
		if err != nil {
			log.Error(err, "Unable to complete migration")
			os.Exit(1)
		}

		// Setup all Controllers
		if err := controller.AddToManager(mgr, operatorCfg); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
	}()

	if err := InitConfig(operatorNamespace, operatorCfg, mgr.GetAPIReader()); err != nil {
		log.Error(err, "Invalid configuration")
		time.Sleep(15 * time.Second)
		return err
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if blocking {
		if err := mgr.Start(signalHandler); err != nil {
			log.Error(err, "Manager exited non-zero")
			return err
		}
	} else {
		go mgr.Start(signalHandler)
	}

	return nil
}

//go:generate counterfeiter -o mocks/reader.go -fake-name Reader . Reader

type Reader interface {
	client.Reader
}

// InitConfig initializes the passed in config by overriding values from environment variable
// or config map if set
func InitConfig(namespace string, cfg *oconfig.Config, client client.Reader) error {
	// Read from config map if it exists otherwise return default values
	err := oconfig.LoadFromConfigMap(
		types.NamespacedName{Name: "operator-config", Namespace: namespace},
		"config.yaml",
		client,
		&cfg.Operator,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get 'config.yaml' from 'ibp-operator' config map")
	}

	clusterType := os.Getenv("CLUSTERTYPE")
	offeringType, err := offering.GetType(clusterType)
	if err != nil {
		return err
	}
	cfg.Offering = offeringType

	log.Info(fmt.Sprintf("Operator configured for cluster type '%s'", cfg.Offering))

	if cfg.Operator.Versions == nil {
		return errors.New("no default images defined")
	}

	if cfg.Operator.Versions.CA == nil {
		return errors.New("no default CA images defined")
	}

	if cfg.Operator.Versions.Peer == nil {
		return errors.New("no default Peer images defined")
	}

	if cfg.Operator.Versions.Orderer == nil {
		return errors.New("no default Orderer images defined")
	}

	return nil
}

func GetOperatorNamespace() (string, error) {
	operatorNamespace := os.Getenv("OPERATOR_NAMESPACE")
	if operatorNamespace == "" {
		return "", fmt.Errorf("OPERATOR_NAMESPACE not found")
	}

	return operatorNamespace, nil
}
func CheckForFixPacks(config *rest.Config, operatornamespace string) {
	clientset, err := kubernetes.NewForConfig(config)

	// Create a dynamic client
	dynamicClient := dynamic.NewForConfigOrDie(config)

	// Define your custom resource type
	//customResourceName := "ibpconsoles"
	//customResourceNamespace := "ibmsupport"
	gvr := schema.GroupVersionResource{
		Group:    "ibp.com",
		Version:  "v1beta1",
		Resource: "ibpconsoles",
	}

	// Retrieve the list of objects in your custom resource
	list, err := dynamicClient.Resource(gvr).Namespace(operatornamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	var consoleObjectName string
	// Print the names of all objects
	for _, obj := range list.Items {

		consoleObjectName = obj.GetName()
		// If you want to do something with the object, you can access it here

	}
	log.Info(fmt.Sprintf("Latest Console Tag is %s", console.GetImages().ConsoleTag))

	// get the console deployment here

	// Retrieve the deployment
	deployment, err := clientset.AppsV1().Deployments(operatornamespace).Get(context.TODO(), consoleObjectName, v1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	existingConsoleDeploymentImageTag := strings.Split(deployment.Spec.Template.Spec.Containers[0].Image, ":")[1]

	log.Info(fmt.Sprintf("Operator Binary Console Tag is %s and current Console Deployment tag is %s", console.GetImages().ConsoleTag, existingConsoleDeploymentImageTag))

	//if the latest console deployment tag and operator binary latest console tag are not same, then we will delete the below two configmaps
	if console.GetImages().ConsoleTag != existingConsoleDeploymentImageTag {
		log.Info(fmt.Sprintf("Will Start Applying the Fixpacks Existing Version %s to New Version %s ", existingConsoleDeploymentImageTag, console.GetImages().ConsoleTag))

		// set the webhook image here as well
		// Specify deployment namespace and name
		namespace := "ibm-hlfsupport-infra"
		deploymentName := "ibm-hlfsupport-webhook"

		// Retrieve the deployment
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, v1.GetOptions{})
		if err != nil {
			panic(err.Error())
		}
		existingwebhookimage := strings.Split(deployment.Spec.Template.Spec.Containers[0].Image, ":")[0]
		existingwebhookimage = existingwebhookimage + ":" + console.GetImages().ConsoleTag

		deployment.Spec.Template.Spec.Containers[0].Image = existingwebhookimage

		// Update the deployment
		_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, v1.UpdateOptions{})
		if err != nil {
			panic(err.Error())
		}

		util.DeleteConfigMapIfExists(clientset, operatornamespace, consoleObjectName+"-console")
		util.DeleteConfigMapIfExists(clientset, operatornamespace, consoleObjectName+"-deployer")

	} else {
		log.Info("Looks like the operator was restarted...")
	}

}
