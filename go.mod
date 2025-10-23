// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

module github.com/sapcc/archer

go 1.25

require (
	github.com/IBM/pgxpoolprometheus v1.1.2
	github.com/Masterminds/squirrel v1.5.4
	github.com/bcicen/go-haproxy v0.0.0-20210728173702-412d077dabc1
	github.com/databus23/goslo.policy v0.0.0-20250326134918-4afc2c56a903
	github.com/didip/tollbooth/v8 v8.0.1
	github.com/dre1080/recovr v1.0.3
	github.com/f5devcentral/go-bigip v0.0.0-20250928174250-859d6942bc8a
	github.com/georgysavva/scany/v2 v2.1.4
	github.com/getsentry/sentry-go v0.20.0
	github.com/go-co-op/gocron/v2 v2.17.0
	github.com/go-openapi/errors v0.22.3
	github.com/go-openapi/loads v0.23.1
	github.com/go-openapi/runtime v0.29.0
	github.com/go-openapi/spec v0.22.0
	github.com/go-openapi/strfmt v0.24.0
	github.com/go-openapi/swag v0.25.1
	github.com/go-openapi/swag/cmdutils v0.25.1
	github.com/go-openapi/swag/conv v0.25.1
	github.com/go-openapi/validate v0.25.0
	github.com/google/uuid v1.6.0
	github.com/gophercloud/gophercloud/v2 v2.8.0
	github.com/gophercloud/utils/v2 v2.0.0-20250930154317-576cdf6142a7
	github.com/hashicorp/go-uuid v1.0.3
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/iancoleman/strcase v0.3.0
	github.com/jackc/pgerrcode v0.0.0-20250907135507-afb5586c32a6
	github.com/jackc/pgx-logrus v0.0.0-20220919124836-b099d8ce75da
	github.com/jackc/pgx/v5 v5.7.6
	github.com/jedib0t/go-pretty/v6 v6.6.8
	github.com/jessevdk/go-flags v1.6.1
	github.com/jmoiron/sqlx v1.4.0
	github.com/pashagolub/pgxmock/v4 v4.9.0
	github.com/prometheus/client_golang v1.23.2
	github.com/rs/cors v1.11.1
	github.com/sapcc/go-api-declarations v1.17.4
	github.com/sapcc/go-bits v0.0.0-20251023170348-b4bcd3ab3925
	github.com/sethvargo/go-retry v0.3.0
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.11.1
	github.com/vishvananda/netlink v1.3.1
	github.com/vishvananda/netns v0.0.5
	github.com/z0ne-dev/mgx/v2 v2.0.1
	golang.org/x/net v0.46.0
	golang.org/x/sync v0.17.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.24.0 // indirect
	github.com/go-openapi/jsonpointer v0.22.1 // indirect
	github.com/go-openapi/jsonreference v0.21.2 // indirect
	github.com/go-openapi/swag/fileutils v0.25.1 // indirect
	github.com/go-openapi/swag/jsonname v0.25.1 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.1 // indirect
	github.com/go-openapi/swag/loading v0.25.1 // indirect
	github.com/go-openapi/swag/mangling v0.25.1 // indirect
	github.com/go-openapi/swag/netutils v0.25.1 // indirect
	github.com/go-openapi/swag/stringutils v0.25.1 // indirect
	github.com/go-openapi/swag/typeutils v0.25.1 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.1 // indirect
	github.com/go-pkgz/expirable-cache/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobuffalo/logger v1.0.3 // indirect
	github.com/gobuffalo/packd v1.0.0 // indirect
	github.com/gobuffalo/packr/v2 v2.8.0 // indirect
	github.com/gocarina/gocsv v0.0.0-20210516172204-ca9e8a8ddea8 // indirect
	github.com/gofrs/uuid/v5 v5.3.2 // indirect
	github.com/golang-migrate/migrate/v4 v4.19.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/karrick/godirwalk v1.15.3 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/logrusorgru/aurora v0.0.0-20181002194514-a7b3b318ed4e // indirect
	github.com/markbates/errx v1.1.0 // indirect
	github.com/markbates/oncer v1.0.0 // indirect
	github.com/markbates/safe v1.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.1 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/ztrue/tracerr v0.3.0 // indirect
	go.mongodb.org/mongo-driver v1.17.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/term v0.36.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
