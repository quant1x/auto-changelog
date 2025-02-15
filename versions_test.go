package main

import "testing"

func Test_incrVersion(t *testing.T) {
	type args struct {
		v    string
		kind VersionKind
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "patch-1",
			args: args{
				v:    "0.3.4",
				kind: PatchVersion,
			},
			want: "0.3.5",
		},
		{
			name: "minor-1",
			args: args{
				v:    "0.3.4",
				kind: MinorVersion,
			},
			want: "0.3.5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := incrVersion(tt.args.v, tt.args.kind); got != tt.want {
				t.Errorf("incrVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
