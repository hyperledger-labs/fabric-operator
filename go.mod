module github.com/IBM-Blockchain/fabric-operator

go 1.24.3

require (
	github.com/cloudflare/cfssl v1.4.1
	github.com/docker/docker v28.3.3+incompatible
	github.com/go-logr/logr v1.4.1
	github.com/go-logr/zapr v0.4.0
	github.com/go-test/deep v1.0.2
	github.com/gogo/protobuf v1.3.2
	github.com/hyperledger/fabric v0.0.0-20240618194258-7c3876255bf0
	github.com/hyperledger/fabric-ca v1.5.13-0.20240620215654-493a48abb873
	github.com/hyperledger/fabric-lib-go v1.1.2
	github.com/hyperledger/fabric-protos-go v0.3.3
	github.com/imdario/mergo v0.3.12
	github.com/lib/pq v1.8.0
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.3
	github.com/onsi/ginkgo/v2 v2.13.2
	github.com/onsi/gomega v1.31.0
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/operator-framework/operator-lib v0.8.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/viper v1.7.0
	github.com/vrischmann/envconfig v1.3.0
	go.uber.org/zap v1.27.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.5
	k8s.io/apiextensions-apiserver v0.21.5
	k8s.io/apimachinery v0.21.5
	k8s.io/client-go v0.21.5
	k8s.io/code-generator v0.21.5
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/controller-runtime v0.9.0
	sigs.k8s.io/yaml v1.2.0
)

require (
	github.com/IBM/idemix v0.0.2-0.20231011101252-a4feda90f3f7 // indirect
	github.com/IBM/mathlib v0.0.3-0.20231011094432-44ee0eb539da // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/jinzhu/copier v0.3.5 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
)

require (
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	github.com/DataDog/zstd v1.5.6 // indirect
	github.com/IBM/idemix/bccsp/schemes/aries v0.0.0-20231003085036-c4470b87b2d6 // indirect
	github.com/IBM/idemix/bccsp/schemes/weak-bb v0.0.0-20240612072411-114d281b442d // indirect
	github.com/IBM/idemix/bccsp/types v0.0.0-20240612072411-114d281b442d // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/Shopify/sarama v1.20.1 // indirect
	github.com/ale-linux/aries-framework-go/component/kmscrypto v0.0.0-20230817163708-4b3de6d91874 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.13.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/emicklei/go-restful v2.16.0+incompatible // indirect
	github.com/evanphx/json-patch v4.11.0+incompatible // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.3 // indirect
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/certificate-transparency-go v1.0.21 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/grantae/certinfo v0.0.0-20170412194111-59d56a35515b // indirect
	github.com/hashicorp/go-version v1.4.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hyperledger/fabric-amcl v0.0.0-20230602173724-9e02669dceb2 // indirect
	github.com/jmhodges/clock v0.0.0-20160418191101-880ee4c33548 // indirect
	github.com/jmoiron/sqlx v1.3.4 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kilic/bls12-381 v0.1.0 // indirect
	github.com/kisielk/sqlstruct v0.0.0-20201105191214-5f3e10d3ab46 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.18 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc6 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.16.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/sykesm/zap-logfmt v0.0.4 // indirect
	github.com/weppos/publicsuffix-go v0.5.0 // indirect
	github.com/zmap/zcrypto v0.0.0-20190729165852-9051775e6a2e // indirect
	github.com/zmap/zlint v0.0.0-20190806154020-fd021b4cfbeb // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/oauth2 v0.27.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/term v0.30.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/grpc v1.64.1 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/ldap.v2 v2.5.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/component-base v0.21.5 // indirect
	k8s.io/gengo v0.0.0-20201214224949-b6c5ce23f027 // indirect
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo/v2 => github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega => github.com/onsi/gomega v1.19.0
)
