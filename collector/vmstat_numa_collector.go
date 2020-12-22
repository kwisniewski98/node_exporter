// Copyright 2020 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"reflect"
	"strconv"

	"github.com/prometheus/procfs/sysfs"

)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")
)


const vmStatNumaSubsystem = "vmstat_numa"

type vmstatNumaCollector struct {
	metricDescs map[string]*prometheus.Desc
	logger      log.Logger
	fs          sysfs.FS
}

func init() {
	registerCollector("vmstat_numa", defaultDisabled, NewVmstatNumaCollector)
}

// NewVmstatNumaCollector returns a new Collector exposing memory stats.
func NewVmstatNumaCollector(logger log.Logger) (Collector, error) {
	fs, err := sysfs.NewFS(*sysPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open procfs: %w", err)
	}
	return &vmstatNumaCollector{
		metricDescs: map[string]*prometheus.Desc{},
		logger:      logger,
		fs:          fs,
	}, nil
}

func (c *vmstatNumaCollector) Update(ch chan<- prometheus.Metric) error {

	metrics, err := c.fs.VmStatNuma()
	if err != nil {
		return fmt.Errorf("couldn't get NUMA vmstat: %w", err)
	}
	for k, v := range metrics {
		metricStruct := reflect.ValueOf(v)
		typeOfMetricStruct := metricStruct.Type()
		for i := 0; i < metricStruct.NumField(); i++ {
			metricName := ToSnakeCase(typeOfMetricStruct.Field(i).Name)
			desc, ok := c.metricDescs[metricName]
			if !ok {
				desc = prometheus.NewDesc(
					prometheus.BuildFQName(namespace, vmStatNumaSubsystem, metricName),
					fmt.Sprintf("Virtual memory information field %s.", metricName),
					[]string{"node"}, nil)
				c.metricDescs[metricName] = desc
			}
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(metricStruct.Field(i).Uint()),
				strconv.Itoa(k))
		}

	}
	return nil
}

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
