package process

import "testing"

func TestParseStat(t *testing.T) {
	tests := []struct {
		name string
		stat string
		want statData
		err  bool
	}{
		{
			name: "simple",
			stat: "5279 (bash) S 5236 5279 5279 34817 315057 4194304 5317 17286479 7 1175 2 1 25093 13281 20 0 1 0 16462 9506816 1185 18446744073709551615 110941457379328 110941458094025 140720938564912 0 0 0 65536 3686404 1266761467 1 0 0 17 8 0 0 0 0 0 110941458286320 110941458310264 110941863948288 140720938568614 140720938568620 140720938568620 140720938577880 0",
			want: statData{
				pid:  5279,
				ppid: 5236,
				comm: "bash",
			},
		},
		{
			name: "missing cmd",
			stat: "5279 S 5236 5279 5279 34817 315057 4194304 5317 17286479 7 1175 2 1 25093 13281 20 0 1 0 16462 9506816 1185 18446744073709551615 110941457379328 110941458094025 140720938564912 0 0 0 65536 3686404 1266761467 1 0 0 17 8 0 0 0 0 0 110941458286320 110941458310264 110941863948288 140720938568614 140720938568620 140720938568620 140720938577880 0",
			err:  true,
		},
		{
			name: "missing ppid",
			stat: "5279 (bash) S",
			err:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStat(tt.stat)

			if tt.err {
				if err == nil {
					t.Errorf("stat() expected error")
				}
			} else if err != nil {
				t.Errorf("stat() error: %s", err)
			} else if got != tt.want {
				t.Errorf("stat() = %-v, want %-v", got, tt.want)
			}
		})
	}
}
