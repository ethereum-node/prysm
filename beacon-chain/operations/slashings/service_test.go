package slashings

import (
	"context"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func attesterSlashingForValIdx(valIdx uint64) *ethpb.AttesterSlashing {
	return &ethpb.AttesterSlashing{
		Attestation_1: &ethpb.IndexedAttestation{
			AttestingIndices: []uint64{valIdx},
		},
		Attestation_2: &ethpb.IndexedAttestation{
			AttestingIndices: []uint64{valIdx},
		},
	}
}

func pendingSlashingForValIdx(valIdx uint64) *PendingAttesterSlashing {
	return &PendingAttesterSlashing{
		attesterSlashing: attesterSlashingForValIdx(valIdx),
		validatorToSlash: valIdx,
	}
}

func generate0ToNSlashings(n uint64) []*ethpb.AttesterSlashing {
	attesterSlashings := make([]*ethpb.AttesterSlashing, n)
	for i := uint64(0); i < n; i++ {
		attesterSlashings[i] = attesterSlashingForValIdx(i)
	}
	return attesterSlashings
}

func generate0ToNPendingSlashings(n uint64) []*PendingAttesterSlashing {
	pendingAttSlashings := make([]*PendingAttesterSlashing, n)
	for i := uint64(0); i < n; i++ {
		pendingAttSlashings[i] = &PendingAttesterSlashing{
			attesterSlashing: attesterSlashingForValIdx(i),
			validatorToSlash: i,
		}
	}
	return pendingAttSlashings
}

func TestPool_InsertAttesterSlashing(t *testing.T) {
	type fields struct {
		pending  []*PendingAttesterSlashing
		included map[uint64]bool
	}
	type args struct {
		slashings *ethpb.AttesterSlashing
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*PendingAttesterSlashing
	}{
		{
			name: "Empty list",
			fields: fields{
				pending:  make([]*PendingAttesterSlashing, 0),
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: attesterSlashingForValIdx(1),
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: attesterSlashingForValIdx(1),
					validatorToSlash: 1,
				},
			},
		},
		{
			name: "Empty list two validators slashed",
			fields: fields{
				pending:  make([]*PendingAttesterSlashing, 0),
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1, 2},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1, 2},
					},
				},
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2},
						},
					},
					validatorToSlash: 1,
				},
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2},
						},
					},
					validatorToSlash: 2,
				},
			},
		},
		{
			name: "Empty list two validators slashed out od three",
			fields: fields{
				pending:  make([]*PendingAttesterSlashing, 0),
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1, 2, 3},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1, 3},
					},
				},
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2, 3},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 3},
						},
					},
					validatorToSlash: 1,
				},
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2, 3},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 3},
						},
					},
					validatorToSlash: 3,
				},
			},
		},
		{
			name: "Duplicate identical slashing",
			fields: fields{
				pending: []*PendingAttesterSlashing{
					{
						attesterSlashing: attesterSlashingForValIdx(1),
						validatorToSlash: 1,
					},
				},
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: attesterSlashingForValIdx(1),
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: attesterSlashingForValIdx(1),
					validatorToSlash: 1,
				},
			},
		},
		{
			name: "Slashing for exited validator ",
			fields: fields{
				pending:  []*PendingAttesterSlashing{},
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: attesterSlashingForValIdx(2),
			},
			want: []*PendingAttesterSlashing{},
		},
		{
			name: "Slashing for futuristic exited validator ",
			fields: fields{
				pending:  []*PendingAttesterSlashing{},
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: attesterSlashingForValIdx(4),
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: attesterSlashingForValIdx(4),
					validatorToSlash: 4,
				},
			},
		},
		{
			name: "Slashing for slashed validator ",
			fields: fields{
				pending:  []*PendingAttesterSlashing{},
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: attesterSlashingForValIdx(5),
			},
			want: []*PendingAttesterSlashing{},
		},
		{
			name: "Already included",
			fields: fields{
				pending: []*PendingAttesterSlashing{},
				included: map[uint64]bool{
					1: true,
				},
			},
			args: args{
				slashings: attesterSlashingForValIdx(1),
			},
			want: []*PendingAttesterSlashing{},
		},
		{
			name: "Already included",
			fields: fields{
				pending: []*PendingAttesterSlashing{},
				included: map[uint64]bool{
					1: true,
				},
			},
			args: args{
				slashings: attesterSlashingForValIdx(1),
			},
			want: []*PendingAttesterSlashing{},
		},
		{
			name: "Already included",
			fields: fields{
				pending: []*PendingAttesterSlashing{},
				included: map[uint64]bool{
					1: true,
				},
			},
			args: args{
				slashings: attesterSlashingForValIdx(1),
			},
			want: []*PendingAttesterSlashing{},
		},
		{
			name: "Maintains sorted order",
			fields: fields{
				pending: []*PendingAttesterSlashing{
					{
						attesterSlashing: attesterSlashingForValIdx(0),
						validatorToSlash: 0,
					},
					{
						attesterSlashing: attesterSlashingForValIdx(2),
						validatorToSlash: 2,
					},
				},
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: attesterSlashingForValIdx(1),
			},
			want: generate0ToNPendingSlashings(3),
		},
		{
			name: "Already included",
			fields: fields{
				pending: make([]*PendingAttesterSlashing, 0),
				included: map[uint64]bool{
					1: true,
				},
			},
			args: args{
				slashings: attesterSlashingForValIdx(1),
			},
			want: []*PendingAttesterSlashing{},
		},
	}
	ctx := context.Background()
	validators := []*ethpb.Validator{
		{ // 0
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{ // 1
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{ // 2 - Already exited.
			ExitEpoch: 15,
		},
		{ // 3
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{ // 4 - Will be exited.
			ExitEpoch: 17,
		},
		{ // 5 - Slashed.
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				pendingAttesterSlashing: tt.fields.pending,
				included:                tt.fields.included,
			}
			s, err := beaconstate.InitializeFromProtoUnsafe(&p2ppb.BeaconState{Validators: validators})
			if err != nil {
				t.Fatal(err)
			}
			s.SetSlot(16 * params.BeaconConfig().SlotsPerEpoch)
			s.SetSlashings([]uint64{5})
			p.InsertAttesterSlashing(ctx, s, tt.args.slashings)
			if len(p.pendingAttesterSlashing) != len(tt.want) {
				t.Fatalf("Mismatched lengths of pending list. Got %d, wanted %d.", len(p.pendingAttesterSlashing), len(tt.want))
			}
			for i := range p.pendingAttesterSlashing {
				if p.pendingAttesterSlashing[i].validatorToSlash != tt.want[i].validatorToSlash {
					t.Errorf("Pending attester to slash at index %d does not match expected. Got=%v wanted=%v", i, p.pendingAttesterSlashing[i].validatorToSlash, tt.want[i].validatorToSlash)
				}
				if !proto.Equal(p.pendingAttesterSlashing[i].attesterSlashing, tt.want[i].attesterSlashing) {
					t.Errorf("Pending attester slashings at index %d does not match expected. Got=%v wanted=%v", i, p.pendingAttesterSlashing[i], tt.want[i])
				}
			}
		})
	}
}

func TestPool_MarkIncludedAttesterSlashing(t *testing.T) {
	type fields struct {
		pending  []*PendingAttesterSlashing
		included map[uint64]bool
	}
	type args struct {
		slashing *ethpb.AttesterSlashing
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   fields
	}{
		{
			name: "Included, does not exist in pending",
			fields: fields{
				pending: []*PendingAttesterSlashing{
					{
						attesterSlashing: attesterSlashingForValIdx(1),
						validatorToSlash: 1,
					},
				},
				included: make(map[uint64]bool),
			},
			args: args{
				slashing: attesterSlashingForValIdx(3),
			},
			want: fields{
				pending: []*PendingAttesterSlashing{
					{
						attesterSlashing: attesterSlashingForValIdx(1),
						validatorToSlash: 1,
					},
				},
				included: map[uint64]bool{
					3: true,
				},
			},
		},
		{
			name: "Removes from pending list",
			fields: fields{
				pending: []*PendingAttesterSlashing{
					{
						attesterSlashing: attesterSlashingForValIdx(1),
						validatorToSlash: 1,
					},
					{
						attesterSlashing: attesterSlashingForValIdx(2),
						validatorToSlash: 2,
					},
					{
						attesterSlashing: attesterSlashingForValIdx(3),
						validatorToSlash: 3,
					},
				},
				included: map[uint64]bool{
					0: true,
				},
			},
			args: args{
				slashing: attesterSlashingForValIdx(2),
			},
			want: fields{
				pending: []*PendingAttesterSlashing{
					{
						attesterSlashing: attesterSlashingForValIdx(1),
						validatorToSlash: 1,
					},
					{
						attesterSlashing: attesterSlashingForValIdx(3),
						validatorToSlash: 3,
					},
				},
				included: map[uint64]bool{
					0: true,
					2: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				pendingAttesterSlashing: tt.fields.pending,
				included:                tt.fields.included,
			}
			p.MarkIncludedAttesterSlashing(tt.args.slashing)
			if len(p.pendingAttesterSlashing) != len(tt.want.pending) {
				t.Fatalf(
					"Mismatched lengths of pending list. Got %d, wanted %d.",
					len(p.pendingAttesterSlashing),
					len(tt.want.pending),
				)
			}
			for i := range p.pendingAttesterSlashing {
				if !reflect.DeepEqual(p.pendingAttesterSlashing[i], tt.want.pending[i]) {
					t.Errorf("Pending exit at index %d does not match expected. Got=%v wanted=%v", i, p.pendingAttesterSlashing[i], tt.want.pending[i])
				}
			}
			if !reflect.DeepEqual(p.included, tt.want.included) {
				t.Errorf("Included map is not as expected. Got=%v wanted=%v", p.included, tt.want.included)
			}
		})
	}
}

func TestPool_PendingAttesterSlashings(t *testing.T) {
	type fields struct {
		pending []*PendingAttesterSlashing
	}
	type args struct {
		validatorToSlash uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   []*PendingAttesterSlashing
	}{
		{
			name: "Empty list",
			fields: fields{
				pending: []*PendingAttesterSlashing{},
			},
			want: []*PendingAttesterSlashing{},
		},
		{
			name: "All eligible",
			fields: fields{
				pending: generate0ToNPendingSlashings(6),
			},
			want: generate0ToNPendingSlashings(6),
		},
		{
			name: "All eligible, more than max",
			fields: fields{
				pending: generate0ToNPendingSlashings(16),
			},
			want: generate0ToNPendingSlashings(16),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				pendingAttesterSlashing: tt.fields.pending,
			}
			if got := p.PendingAttesterSlashings(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PendingExits() = %v, want %v", got, tt.want)
			}
		})
	}
}
