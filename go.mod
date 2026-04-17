// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

module github.com/sapcc/archer

go 1.26

require (
	github.com/IBM/pgxpoolprometheus v1.1.2
	github.com/Masterminds/squirrel v1.5.4
	github.com/bcicen/go-haproxy v0.0.0-20210728173702-412d077dabc1
	github.com/databus23/goslo.policy v0.0.0-20250326134918-4afc2c56a903
	github.com/didip/tollbooth/v8 v8.0.1
	github.com/dre1080/recovr v1.0.3
	github.com/f5devcentral/go-bigip v0.0.0-20260303071915-79fc4d8a2250
	github.com/georgysavva/scany/v2 v2.1.4
	github.com/getsentry/sentry-go v0.20.0
	github.com/go-co-op/gocron/v2 v2.20.0
	github.com/go-openapi/errors v0.22.7
	github.com/go-openapi/loads v0.23.3
	github.com/go-openapi/runtime v0.29.3
	github.com/go-openapi/spec v0.22.4
	github.com/go-openapi/strfmt v0.26.1
	github.com/go-openapi/swag v0.25.5
	github.com/go-openapi/swag/cmdutils v0.25.5
	github.com/go-openapi/swag/conv v0.25.5
	github.com/go-openapi/validate v0.25.2
	github.com/google/uuid v1.6.0
	github.com/gophercloud/gophercloud/v2 v2.12.0
	github.com/gophercloud/utils/v2 v2.0.0-20260409164906-cd2167a8681a
	github.com/hashicorp/go-uuid v1.0.3
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/iancoleman/strcase v0.3.0
	github.com/jackc/pgerrcode v0.0.0-20250907135507-afb5586c32a6
	github.com/jackc/pgx-logrus v0.0.0-20220919124836-b099d8ce75da
	github.com/jackc/pgx/v5 v5.9.1
	github.com/jedib0t/go-pretty/v6 v6.7.9
	github.com/jessevdk/go-flags v1.6.1
	github.com/jmoiron/sqlx v1.4.0
	github.com/moby/moby/api v1.54.1
	github.com/pashagolub/pgxmock/v5 v5.0.1
	github.com/prometheus/client_golang v1.23.2
	github.com/rs/cors v1.11.1
	github.com/sapcc/go-api-declarations v1.21.0
	github.com/sapcc/go-bits v0.0.0-20260417095247-780e38e22ec4
	github.com/sethvargo/go-retry v0.3.0
	github.com/sirupsen/logrus v1.9.4
	github.com/stretchr/testify v1.11.1
	github.com/testcontainers/testcontainers-go v0.42.0
	github.com/testcontainers/testcontainers-go/modules/postgres v0.42.0
	github.com/vishvananda/netlink v1.3.1
	github.com/vishvananda/netns v0.0.5
	github.com/z0ne-dev/mgx/v2 v2.0.1
	golang.org/x/net v0.53.0
	golang.org/x/sync v0.20.0
	golang.org/x/sys v0.43.0
	golang.org/x/term v0.42.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/analysis v0.24.3 // indirect
	github.com/go-openapi/jsonpointer v0.22.5 // indirect
	github.com/go-openapi/jsonreference v0.21.5 // indirect
	github.com/go-openapi/swag/fileutils v0.25.5 // indirect
	github.com/go-openapi/swag/jsonname v0.25.5 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.5 // indirect
	github.com/go-openapi/swag/loading v0.25.5 // indirect
	github.com/go-openapi/swag/mangling v0.25.5 // indirect
	github.com/go-openapi/swag/netutils v0.25.5 // indirect
	github.com/go-openapi/swag/stringutils v0.25.5 // indirect
	github.com/go-openapi/swag/typeutils v0.25.5 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.5 // indirect
	github.com/go-pkgz/expirable-cache/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gobuffalo/logger v1.0.3 // indirect
	github.com/gobuffalo/packd v1.0.0 // indirect
	github.com/gobuffalo/packr/v2 v2.8.0 // indirect
	github.com/gocarina/gocsv v0.0.0-20210516172204-ca9e8a8ddea8 // indirect
	github.com/gofrs/uuid/v5 v5.4.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/karrick/godirwalk v1.15.3 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/logrusorgru/aurora v0.0.0-20181002194514-a7b3b318ed4e // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/markbates/errx v1.1.0 // indirect
	github.com/markbates/oncer v1.0.0 // indirect
	github.com/markbates/safe v1.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.2.0 // indirect
	github.com/moby/moby/client v0.4.0 // indirect
	github.com/moby/patternmatcher v0.6.1 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/shirou/gopsutil/v4 v4.26.3 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/ztrue/tracerr v0.3.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel v1.41.0 // indirect
	go.opentelemetry.io/otel/metric v1.41.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.41.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
