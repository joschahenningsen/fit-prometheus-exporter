package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"reflect"
	"time"

	"github.com/castai/promwrite"
	"github.com/tormoder/fit"
)

func main() {
	f := flag.String("f", "", "path to the FIT file")
	endpoint := flag.String("prometheus", "http://localhost:9090/api/v1/write", "prometheus remote write endpoint")
	flag.Parse()
	if f == nil {
		flag.Usage()
		return
	}

	data, err := os.ReadFile(*f)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Decode the FIT file data
	fit, err := fit.Decode(bytes.NewReader(data))
	if err != nil {
		fmt.Println(err)
		return
	}
	a, _ := fit.Activity()
	result := make(map[string]map[string]any)
	for _, record := range a.Records {
		key := record.Timestamp.Format(time.DateTime + ".000")
		result[key] = make(map[string]any)
		r := *record
		t := reflect.TypeOf(r)
		val := reflect.ValueOf(r)
		fields := reflect.VisibleFields(t)
		for _, field := range fields {
			fieldVal := val.FieldByName(field.Name)
			if !isFieldValid(fieldVal.Interface()) {
				continue
			}
			result[key][field.Name] = fieldVal.Interface()
			pushMetric(*endpoint, field.Name, record.Timestamp, toFloat64(fieldVal.Interface()))
		}
	}
}

func toFloat64(value interface{}) float64 {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	default:
		return -1
	}
}

func isFieldValid(input any) bool {
	switch input.(type) {
	case int8:
		return input.(int8) != math.MaxInt8
	case int16:
		return input.(int16) != math.MaxInt16
	case int32:
		return input.(int32) != math.MaxInt32
	case int64:
		return input.(int64) != math.MaxInt64
	case uint8:
		return input.(uint8) != math.MaxUint8
	case uint16:
		return input.(uint16) != math.MaxUint16
	case uint32:
		return input.(uint32) != math.MaxUint32
	case uint64:
		return input.(uint64) != math.MaxUint64
	}
	return false
}
func pushMetric(endpoint, name string, time time.Time, val float64) {
	client := promwrite.NewClient(endpoint)
	req := &promwrite.WriteRequest{
		TimeSeries: []promwrite.TimeSeries{
			{
				Labels: []promwrite.Label{
					{
						Name:  "__name__",
						Value: "garmin_" + name,
					},
				},
				Sample: promwrite.Sample{
					Time:  time,
					Value: val,
				},
			},
		},
	}
	resp, err := client.Write(context.Background(), req)
	if err != nil {
		fmt.Println(err, resp)
	}
}
