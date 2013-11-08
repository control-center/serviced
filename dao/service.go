package dao

func (a *Service) Equals(b *Service) bool {
	if a.Id != b.Id {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.Context != b.Context {
		return false
	}
	if a.Startup != b.Startup {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.Instances != b.Instances {
		return false
	}
	if a.ImageId != b.ImageId {
		return false
	}
	if a.PoolId != b.PoolId {
		return false
	}
	if a.DesiredState != b.DesiredState {
		return false
	}
	if a.Launch != b.Launch {
		return false
	}
	if a.ParentServiceId != b.ParentServiceId {
		return false
	}
	if a.CreatedAt != b.CreatedAt {
		return false
	}
	if a.UpdatedAt != b.CreatedAt {
		return false
	}
	return true
}
