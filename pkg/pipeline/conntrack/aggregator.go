/*
 * Copyright (C) 2022 IBM, Inc.
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
 */

package conntrack

import (
	"fmt"
	"math"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	"github.com/netobserv/flowlogs-pipeline/pkg/config"
	"github.com/netobserv/flowlogs-pipeline/pkg/pipeline/utils"
	log "github.com/sirupsen/logrus"
)

type aggregator interface {
	addField(conn connection)
	update(conn connection, flowLog config.GenericMap, d direction)
}

type aggregateBase struct {
	inputField  string
	outputField string
	splitAB     bool
	initVal     float64
}

type aggregateSum struct {
	aggregateBase
}

type aggregateCount struct {
	aggregateBase
}

type aggregateMin struct {
	aggregateBase
}

type aggregateMax struct {
	aggregateBase
}

// TODO: Should return a pointer?
func NewAggregator(of api.OutputField) (aggregator, error) {
	if of.Name == "" {
		return nil, fmt.Errorf("empty name %v", of)
	}
	var inputField string
	if of.Input != "" {
		inputField = of.Input
	} else {
		inputField = of.Name
	}
	aggBase := aggregateBase{inputField: inputField, outputField: of.Name, splitAB: of.SplitAB}
	var agg aggregator
	switch of.Operation {
	case "sum":
		aggBase.initVal = 0
		agg = aggregateSum{aggBase}
	case "count":
		aggBase.initVal = 0
		agg = aggregateCount{aggBase}
	case "min":
		aggBase.initVal = math.MaxFloat64
		agg = aggregateMin{aggBase}
	case "max":
		aggBase.initVal = -math.MaxFloat64
		agg = aggregateMax{aggBase}
	default:
		return nil, fmt.Errorf("unknown operation: %q", of.Operation)
	}
	return agg, nil
}

func (agg aggregateBase) getOutputField(d direction) string {
	outputField := agg.outputField
	if agg.splitAB {
		switch d {
		case dirAB:
			outputField += "_AB"
		case dirBA:
			outputField += "_BA"
		default:
			log.Panicf("splitAB aggregator %v cannot determine outputField because direction is missing. Check configuration.", outputField)
		}
	}
	return outputField
}

func (agg aggregateBase) addField(conn connection) {
	if agg.splitAB {
		conn.addAgg(agg.getOutputField(dirAB), agg.initVal)
		conn.addAgg(agg.getOutputField(dirBA), agg.initVal)
	} else {
		conn.addAgg(agg.getOutputField(dirNA), agg.initVal)
	}
}

func (agg aggregateBase) getInputFieldValue(flowLog config.GenericMap) (float64, error) {
	rawValue, ok := flowLog[agg.inputField]
	if !ok {
		return 0, fmt.Errorf("missing field %v", agg.inputField)
	}
	floatValue, err := utils.ConvertToFloat64(rawValue)
	if err != nil {
		return 0, fmt.Errorf("cannot convert %v to float64: %w", rawValue, err)
	}
	return floatValue, nil
}

func (agg aggregateSum) update(conn connection, flowLog config.GenericMap, d direction) {
	outputField := agg.getOutputField(d)
	v, err := agg.getInputFieldValue(flowLog)
	if err != nil {
		log.Errorf("error updating connection %v: %v", string(conn.Hash().hashTotal), err)
		return
	}
	conn.updateAggValue(outputField, func(curr float64) float64 {
		return curr + v
	})
}

func (agg aggregateCount) update(conn connection, flowLog config.GenericMap, d direction) {
	outputField := agg.getOutputField(d)
	conn.updateAggValue(outputField, func(curr float64) float64 {
		return curr + 1
	})
}

func (agg aggregateMin) update(conn connection, flowLog config.GenericMap, d direction) {
	outputField := agg.getOutputField(d)
	v, err := agg.getInputFieldValue(flowLog)
	if err != nil {
		log.Errorf("error updating connection %v: %v", string(conn.Hash().hashTotal), err)
		return
	}

	conn.updateAggValue(outputField, func(curr float64) float64 {
		return math.Min(curr, v)
	})
}

func (agg aggregateMax) update(conn connection, flowLog config.GenericMap, d direction) {
	outputField := agg.getOutputField(d)
	v, err := agg.getInputFieldValue(flowLog)
	if err != nil {
		log.Errorf("error updating connection %v: %v", string(conn.Hash().hashTotal), err)
		return
	}

	conn.updateAggValue(outputField, func(curr float64) float64 {
		return math.Max(curr, v)
	})
}
