package efgs

import (
	"encoding/json"
	"fmt"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"io/ioutil"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestToDiagnosisKey(t *testing.T) {
	type args struct {
		symptomsDate time.Time
		key          *keyserverapi.ExposureKey
	}

	parsed := func(start string) time.Time {
		t, err := time.Parse("2.1.2006", start)
		if err != nil {
			panic(err)
		}
		return t
	}

	newExpKey := func(start string) *efgsapi.ExpKey {
		interval := parsed(start).Unix() / 600
		return &efgsapi.ExpKey{Key: "YQ==", IntervalNumber: int32(interval), IntervalCount: 144, TransmissionRisk: 2}
	}

	newDiagKey := func(start string, dsos int32) *efgsapi.DiagnosisKey {
		interval := parsed(start).Unix() / 600
		return &efgsapi.DiagnosisKey{
			KeyData:                    []byte{97},
			RollingStartIntervalNumber: uint32(interval),
			RollingPeriod:              144,
			TransmissionRiskLevel:      2,
			VisitedCountries:           []string{"DE"},
			Origin:                     "CZ",
			ReportType:                 efgsapi.ReportType_CONFIRMED_TEST,
			DaysSinceOnsetOfSymptoms:   dsos,
		}
	}

	tests := []struct {
		name string
		args args
		want *efgsapi.DiagnosisKey
	}{
		{
			name: "basic-after1",
			args: args{
				symptomsDate: parsed("10.12.2020"),
				key:          newExpKey("10.12.2020"),
			},
			want: newDiagKey("10.12.2020", 0),
		},
		{
			name: "basic-after2",
			args: args{
				symptomsDate: parsed("10.12.2020"),
				key:          newExpKey("9.12.2020"),
			},
			want: newDiagKey("9.12.2020", -1),
		},
		{
			name: "basic-after3",
			args: args{
				symptomsDate: parsed("10.12.2020"),
				key:          newExpKey("5.12.2020"),
			},
			want: newDiagKey("5.12.2020", -5),
		},
		{
			name: "basic-before1",
			args: args{
				symptomsDate: parsed("9.12.2020"),
				key:          newExpKey("10.12.2020"),
			},
			want: newDiagKey("10.12.2020", 1),
		},
		{
			name: "basic-before2",
			args: args{
				symptomsDate: parsed("5.12.2020"),
				key:          newExpKey("10.12.2020"),
			},
			want: newDiagKey("10.12.2020", 5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToDiagnosisKey(tt.args.symptomsDate, tt.args.key, "CZ", []string{"DE"}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToDiagnosisKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitKeysProp(t *testing.T) {
	// In real world, some states upload even same-day-TEKs which cause troubles with batches alignment.
	// Here we generate random intervals and period which is unaligned by design.
	intervalNumberGen := gen.Int32Range(250000, 450000)
	intervalCountGen := gen.Int32Range(1, 144)

	keyGen := gopter.CombineGens(gen.AlphaString(), intervalNumberGen, intervalCountGen).Map(func(v *gopter.GenResult) efgsapi.ExpKey {
		key := efgsapi.ExpKey{}

		for i, value := range v.Result.([]interface{}) {
			switch i {
			case 0:
				key.Key = value.(string)
			case 1:
				key.IntervalNumber = value.(int32)
			case 2:
				key.IntervalCount = value.(int32)
			}
		}

		return key
	})

	keysCountGen := gen.UInt16().SuchThat(func(v uint16) bool { return v > 0 }) // means 1 - 65535 which is a very real number of keys we can get

	properties := gopter.NewProperties(nil)

	properties.Property("splits random batches to valid", prop.ForAll(
		func(keysCount uint16, batchSize int, maxOverlapping int) bool {
			var keys []efgsapi.ExpKey
			for i := 0; i < int(keysCount); i++ {
				key, b := keyGen.Sample()
				if !b {
					panic(":-(")
				}
				keys = append(keys, key.(efgsapi.ExpKey))
			}

			batches := splitKeys(keys, batchSize, maxOverlapping)

			for _, batch := range batches {
				if err := checkKeyServerBatch(batch, batchSize, maxOverlapping); err != nil {
					t.Logf("Failing: %v %v %v %+v\n", keysCount, batchSize, maxOverlapping, keys)
					panic(err)
				}
			}

			return true
		},
		keysCountGen.WithLabel("keys"),
		gen.IntRange(1, 1000).WithLabel("batchSize"),
		gen.IntRange(1, 1000).WithLabel("maxOverlapping"),
	))

	properties.TestingRun(t)
}

func TestSplitKeys(t *testing.T) {
	type args struct {
		keys           []efgsapi.ExpKey
		batchSize      int
		maxOverlapping int
	}

	tests := []struct {
		name            string
		args            args
		wantChunks      [][]efgsapi.ExpKey
		wantMatchResult bool
		wantChunksCount int
	}{
		{
			name: "basic",
			args: args{
				keys: []efgsapi.ExpKey{
					{Key: "a", IntervalNumber: 10, IntervalCount: 0, TransmissionRisk: 0},
					{Key: "b", IntervalNumber: 0, IntervalCount: 0, TransmissionRisk: 0},
					{Key: "c", IntervalNumber: 10, IntervalCount: 0, TransmissionRisk: 0},
					{Key: "d", IntervalNumber: 10, IntervalCount: 2, TransmissionRisk: 0},
					{Key: "e", IntervalNumber: 0, IntervalCount: 2, TransmissionRisk: 0},
					{Key: "f", IntervalNumber: 0, IntervalCount: 2, TransmissionRisk: 0},
					{Key: "g", IntervalNumber: 0, IntervalCount: 2, TransmissionRisk: 0},
					{Key: "h", IntervalNumber: 0, IntervalCount: 2, TransmissionRisk: 0},
				},
				batchSize:      50,
				maxOverlapping: 2,
			},
			wantChunks: [][]efgsapi.ExpKey{
				{
					{Key: "b", IntervalNumber: 0, IntervalCount: 0, TransmissionRisk: 0},
					{Key: "g", IntervalNumber: 0, IntervalCount: 2, TransmissionRisk: 0},
				},
				{
					{Key: "e", IntervalNumber: 0, IntervalCount: 2, TransmissionRisk: 0},
					{Key: "f", IntervalNumber: 0, IntervalCount: 2, TransmissionRisk: 0},
					{Key: "d", IntervalNumber: 10, IntervalCount: 2, TransmissionRisk: 0},
				},
				{
					{Key: "h", IntervalNumber: 0, IntervalCount: 2, TransmissionRisk: 0},
					{Key: "c", IntervalNumber: 10, IntervalCount: 0, TransmissionRisk: 0},
					{Key: "a", IntervalNumber: 10, IntervalCount: 0, TransmissionRisk: 0},
				},
			},
			wantMatchResult: true,
			wantChunksCount: 3,
		},
		{
			name: "efgsDownloadedKeys",
			args: args{
				keys:           loadTestingKeys("aligned"),
				batchSize:      50,
				maxOverlapping: 15,
			},
			wantChunks:      nil,
			wantMatchResult: false,
			wantChunksCount: 101,
		},
		{
			name: "efgsDownloadedKeysUnalignedSameDay",
			args: args{
				keys:           loadTestingKeys("unaligned-same-day"),
				batchSize:      50,
				maxOverlapping: 15,
			},
			wantChunks:      nil,
			wantMatchResult: false,
			wantChunksCount: 132,
		},
		{
			name: "efgsDownloadedKeysUnaligned",
			args: args{
				keys:           loadTestingKeys("unaligned"),
				batchSize:      50,
				maxOverlapping: 15,
			},
			wantChunks:      nil,
			wantMatchResult: false,
			wantChunksCount: 139,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChunks := splitKeys(tt.args.keys, tt.args.batchSize, tt.args.maxOverlapping)

			for _, chunk := range gotChunks {
				if err := checkKeyServerBatch(chunk, tt.args.batchSize, tt.args.maxOverlapping); err != nil {
					t.Errorf("splitKeys() check error = %+v", err)
					return
				}
			}

			if len(gotChunks) != tt.wantChunksCount {
				t.Errorf("splitKeys() check count error; got %v, want %v", len(gotChunks), tt.wantChunksCount)
				return
			}

			t.Logf("Chunks: %v", len(gotChunks))

			if tt.wantMatchResult && !reflect.DeepEqual(gotChunks, tt.wantChunks) {
				t.Errorf("splitKeys() gotChunks = %+v, want %+v", gotChunks, tt.wantChunks)
			}
		})
	}
}

func loadTestingKeys(suffix string) (efgsDownloadedKeys []efgsapi.ExpKey) {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("../../../test/data/efgs-downloaded-keys-%v.json", suffix))
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(bytes, &efgsDownloadedKeys); err != nil {
		panic(err)
	}

	return efgsDownloadedKeys
}

// This is more-or-less copied from https://github.com/google/exposure-notifications-server/blob/release-0.9/internal/publish/model/exposure_model.go#L418
// because that is the source of truth in this case.
func checkKeyServerBatch(entities []efgsapi.ExpKey, maxSize int, maxOverlapping int) error {
	batchSize := len(entities)

	if batchSize == 0 {
		return fmt.Errorf("Empty batch")
	}

	if batchSize > maxSize {
		return fmt.Errorf("Too big batch; %v > %v", batchSize, maxSize)
	}

	sort.Slice(entities, func(i int, j int) bool {
		if entities[i].IntervalNumber == entities[j].IntervalNumber {
			return entities[i].IntervalCount < entities[j].IntervalCount
		}
		return entities[i].IntervalNumber < entities[j].IntervalNumber
	})

	// Running count of start intervals.
	startIntervals := make(map[int32]int)
	lastInterval := entities[0].IntervalNumber
	nextInterval := entities[0].IntervalNumber + entities[0].IntervalCount

	for _, ex := range entities {
		// Relies on the default value of 0 for the map value type.
		startIntervals[ex.IntervalNumber] = startIntervals[ex.IntervalNumber] + 1

		if ex.IntervalNumber == lastInterval {
			// OK, overlaps by start interval. But move out the nextInterval
			nextInterval = ex.IntervalNumber + ex.IntervalCount
			continue
		}

		if ex.IntervalNumber < nextInterval {
			return fmt.Errorf("Exposure keys have non aligned overlapping intervals. %v overlaps with previous key that is good from %v to %v", ex.IntervalNumber, lastInterval, nextInterval)
		}
		// OK, current key starts at or after the end of the previous one. Advance both variables.
		lastInterval = ex.IntervalNumber
		nextInterval = ex.IntervalNumber + ex.IntervalCount
	}

	for k, v := range startIntervals {
		if v > maxOverlapping {
			return fmt.Errorf("Too many overlapping keys for start interval %v; want <= %v, got %v", k, maxOverlapping, v)
		}
	}

	return nil
}
