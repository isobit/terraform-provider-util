# Example of a resource that will be protected from destruction.
resource "terraform_data" "protected" {}

# This indestructible resources protects the resource from destruction by
# depending on it; if terraform tries to destroy the proected resource, it will
# destroy util_indestructible.this first, which will fail unless allow_destroy
# is set to true.
resource "util_indestructible" "this" {
	# The protected value can be any attribute of the protected resource that
	# would change under replacement. The resource can be also be protected using
	# an explicit "depends_on", but in that case the resource also needs to have
	# "prevent_destroy = true" in the lifecycle configuration to prevent
	# destruction during replacement. Using protected_values simultaneously
	# implies the dependency relationship, and protects against destruction
	# during replacement by causing the indestructible resource itself to also be
	# replaced when the protected resource is replaced.
	protected_value = terraform_data.protected.id
	# Custom messages are optional, but helpful to explain why it is important to
	# avoid destroying the protected resources.
	error_message = "Destroying terraform_data.protected will cause major issues, please don't destroy it!"
}
