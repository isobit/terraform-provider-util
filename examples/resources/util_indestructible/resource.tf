# Example of a resource that will be protected from destruction.
resource "terraform_data" "protected" {
	# It is still important to configure the lifecycle with prevent_destroy to
	# avoid destruction if the plan requires replacement. Note that
	# prevent_destroy alone does not prevent destruction when the entire config
	# is removed; util_indestructible.this is needed to prevent that.
	lifecycle {
		prevent_destroy = true
	}
}

# This indestructible resources protects the resource from destruction by
# depending on it; if terraform tries to destroy the proected resource, it will
# destroy util_indestructible.this first, which will fail unless allow_destroy
# is set to true.
resource "util_indestructible" "this" {
	depends_on = [
		terraform_data.protected,
	]
	# Custom messages are optional, but helpful to explain why it is important to
	# avoid destroying the protected resources.
	error_message = "Destroying terraform_data.protected will cause major issues, please don't destroy it!"
}
