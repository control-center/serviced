package commons

import (
	"reflect"
	"testing"
)

type ImageIDTest struct {
	in        string
	out       *ImageID
	roundtrip string
}

var imgidtests = []ImageIDTest{
	// no host
	{
		"dobbs/sierramadre",
		&ImageID{
			User: "dobbs",
			Repo: "sierramadre",
		},
		"",
	},
	// no host top-level
	{
		"sierramadre",
		&ImageID{
			Repo: "sierramadre",
		},
		"",
	},
	// no host tagged
	{
		"dobbs/sierramadre:1925",
		&ImageID{
			User: "dobbs",
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// no host top-level tagged
	{
		"sierramadre:1925",
		&ImageID{
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// host top-level
	{
		"warner.bros/sierramadre",
		&ImageID{
			Host: "warner.bros",
			Repo: "sierramadre",
		},
		"",
	},
	// host top-level tagged
	{
		"warner.bros/sierramadre:1925",
		&ImageID{
			Host: "warner.bros",
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// host:port top-level
	{
		"warner.bros:1948/sierramadre",
		&ImageID{
			Host: "warner.bros",
			Port: 1948,
			Repo: "sierramadre",
		},
		"",
	},
	// host:port top-level tagged
	{
		"warner.bros:1948/sierramadre:1925",
		&ImageID{
			Host: "warner.bros",
			Port: 1948,
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// host
	{
		"warner.bros/dobbs/sierramadre",
		&ImageID{
			Host: "warner.bros",
			User: "dobbs",
			Repo: "sierramadre",
		},
		"",
	},
	// host tagged
	{
		"warner.bros/dobbs/sierramadre:1925",
		&ImageID{
			Host: "warner.bros",
			User: "dobbs",
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// host:port
	{
		"warner.bros:1948/dobbs/sierramadre",
		&ImageID{
			Host: "warner.bros",
			Port: 1948,
			User: "dobbs",
			Repo: "sierramadre",
		},
		"",
	},
	// host:port tagged
	{
		"warner.bros:1948/dobbs/sierramadre:1925",
		&ImageID{
			Host: "warner.bros",
			Port: 1948,
			User: "dobbs",
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// short hostname:port uuid name
	{
		"warner:1948/543c56d1-2510-cd37-c0f4-cab544df985d_sierramadre",
		&ImageID{
			Host: "warner",
			Port: 1948,
			UUID: "543c56d1-2510-cd37-c0f4-cab544df985d",
			Repo: "sierramadre",
		},
		"",
	},
	// Docker ParseRepositoryTag testcase
	{
		"localhost.localdomain:5000/samalba/hipache:latest",
		&ImageID{
			Host: "localhost.localdomain",
			Port: 5000,
			User: "samalba",
			Repo: "hipache",
			Tag:  "latest",
		},
		"",
	},
}

func DoTest(t *testing.T, parse func(string) (*ImageID, error), name string, tests []ImageIDTest) {
	for _, tt := range tests {
		imgid, err := parse(tt.in)
		if err != nil {
			t.Errorf("%s(%q) returned error %s", name, tt.in, err)
		}
		if !reflect.DeepEqual(imgid, tt.out) {
			t.Errorf("%s(%q):\n\thave %v\n\twant %v\n",
				name, tt.in, imgid, tt.out)
		}
		if tt.in != imgid.String() {
			t.Errorf("%s(%q):\n\thave %v\n\twant %v\n",
				name, tt.in, imgid.String(), tt.in)
		}
	}
}

func TestParse(t *testing.T) {
	DoTest(t, ParseImageID, "Parse", imgidtests)
}

func TestString(t *testing.T) {
	iid, err := ParseImageID("warner.bros:1948/dobbs/sierramadre:1925")
	if err != nil {
		t.Fatal(err)
	}

	if iid.String() != "warner.bros:1948/dobbs/sierramadre:1925" {
		t.Errorf("expecting: warner.bros:1948/dobbs/sierramadre:1925, got %s\n", iid.String())
	}
}

func TestBogusTag(t *testing.T) {
	_, err := ParseImageID("sierramadre:feature/classic")
	if err == nil {
		t.Fatal("expected failure, bad tag")
	}
}

func TestValidateInvalid(t *testing.T) {
	iid := &ImageID{
		Host: "warner.bros",
		Port: 1948,
		User: "d0bbs",
		Repo: "sierramadre",
		Tag:  "feature",
	}

	if iid.Validate() {
		t.Fatal("expecting failure, bad user")
	}
}

func TestValidateValid(t *testing.T) {
	iid := &ImageID{
		Repo: "sierramadre",
		Tag:  "feature",
	}

	if !iid.Validate() {
		t.Fatal("expecting success: ", iid.String())
	}
}
