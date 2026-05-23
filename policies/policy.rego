package gateway.authz

default allow = false

# Public routes are always allowed by the OPA policy
allow {
    input.path == "/auth/register-device"
}

# Edge transfer initiation constraints (standard values under $10,000)
allow {
    input.path == "/transfer/initiate"
    input.method == "POST"
    
    # Must have a valid user subject
    input.claims.sub != ""
    
    # Enforce standard transaction safety limits
    amount_cents_under_limit(input.body.amount_cents)
}

# Edge high-value transfer (>= $10,000) requires a specific high-value scope
allow {
    input.path == "/transfer/initiate"
    input.method == "POST"
    
    input.claims.sub != ""
    
    input.body.amount_cents >= 1000000
    has_scope(input.claims.scopes, "transfer:high_value")
}

# Intranet execution rules
allow {
    input.path == "/private/transfers"
    input.method == "POST"
    
    # Private routes require intranet scopes
    has_scope(input.claims.scopes, "transfer:execute")
}

# Helper functions
amount_cents_under_limit(cents) {
    cents < 1000000
}

has_scope(scopes, required) {
    scopes[_] == required
}
