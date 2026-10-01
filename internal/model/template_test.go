package model

import (
	"testing"
)

func TestTemplateHashDeterministic(t *testing.T) {
	template := PodTemplateSpec{
		Containers: []ContainerSpec{
			{
				Name:  "nginx",
				Image: "nginx:latest",
			},
		},
	}

	fistHash := TemplateHash(template)
	secondHash := TemplateHash(template)

	if fistHash != secondHash {
		t.Errorf("Expected deterministic hash, but got different values: %s and %s", fistHash, secondHash)
	}
}

func TestTemplateHashChangesWithDifferentImages(t *testing.T) {
	first := PodTemplateSpec{
		Containers: []ContainerSpec{
			{
				Name:  "nginx",
				Image: "nginx:latest",
			},
		},
	}
	second := PodTemplateSpec{
		Containers: []ContainerSpec{
			{
				Name:  "nginx",
				Image: "nginx:1.29",
			},
		},
	}
	firstHash := TemplateHash(first)
	secondHash := TemplateHash(second)

	if firstHash == secondHash {
		t.Errorf("Expected different hash for different images, but got the same value: %s", firstHash)
	}
}

func TestTemplateHashChangesWithCommand(t *testing.T) {
	first := PodTemplateSpec{
		Containers: []ContainerSpec{
			{
				Name:    "nginx",
				Image:   "nginx:latest",
				Command: []string{"nginx", "-g", "daemon off;"},
			},
		},
	}
	second := PodTemplateSpec{
		Containers: []ContainerSpec{
			{
				Name:    "nginx",
				Image:   "nginx:latest",
				Command: []string{"nginx", "-g", "daemon off;", "-c", "/etc/nginx/nginx.conf"},
			},
		},
	}
	firstHash := TemplateHash(first)
	secondHash := TemplateHash(second)

	if firstHash == secondHash {
		t.Errorf("Expected different hash for different commands, but got the same value: %s", firstHash)
	}
}

func TestTemplateHashChangesWithArgs(t *testing.T) {
	first := PodTemplateSpec{
		Containers: []ContainerSpec{
			{
				Name:  "nginx",
				Image: "nginx:latest",
				Args:  []string{"-g", "daemon off;"},
			},
		},
	}
	second := PodTemplateSpec{
		Containers: []ContainerSpec{
			{
				Name:  "nginx",
				Image: "nginx:latest",
				Args:  []string{"-g", "daemon off;", "-c", "/etc/nginx/nginx.conf"},
			},
		},
	}
	firstHash := TemplateHash(first)
	secondHash := TemplateHash(second)

	if firstHash == secondHash {
		t.Errorf("Expected different hash for different args, but got the same value: %s", firstHash)
	}
}
