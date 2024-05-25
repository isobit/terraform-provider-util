terraform {
	required_providers {
		pscustom = {
			source = "registry.terraform.io/isobit/util"
		}
	}
}

provider "util" {}

resource "terraform_data" "test" {
	triggers_replace = ["2"]
	provisioner "local-exec" {
		when    = destroy
		command = "echo 'Destroy-time provisioner'"
	}
}

resource "util_indestructable" "test" {
	depends_on = [
		terraform_data.test,
	]
	# allow_destroy = true
}
