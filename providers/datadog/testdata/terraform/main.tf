terraform {
  required_providers {
    datadog = {
      source  = "DataDog/datadog"
      version = "~> 3.0"
    }
  }
}

variable "dd_api_key" {
  description = "Datadog API key"
  type        = string
  sensitive   = true
}

variable "dd_app_key" {
  description = "Datadog Application key"
  type        = string
  sensitive   = true
}

variable "dd_site" {
  description = "Datadog site (e.g., datadoghq.com, datadoghq.eu)"
  type        = string
  default     = "datadoghq.com"
}

provider "datadog" {
  api_key = var.dd_api_key
  app_key = var.dd_app_key
  api_url = "https://api.${var.dd_site}/"
}

# --- User ---

resource "datadog_user" "test" {
  email = "mql-test-user@mondoo.com"
  name  = "MQL Test User"
  roles = []
}

# --- Role ---

resource "datadog_role" "test" {
  name = "mql-test-role"
}

# --- Monitor ---

resource "datadog_monitor" "test_cpu" {
  name    = "mql-test-cpu-monitor"
  type    = "metric alert"
  message = "CPU usage is high on {{host.name}}. @mql-test"
  query   = "avg(last_5m):avg:system.cpu.user{*} > 90"

  monitor_thresholds {
    critical = 90
    warning  = 80
  }

  notify_no_data = true
  tags           = ["mql-test", "env:test"]
}

resource "datadog_monitor" "test_log" {
  name    = "mql-test-log-monitor"
  type    = "log alert"
  message = "Error rate is high. @mql-test"
  query   = "logs(\"status:error service:mql-test\").index(\"*\").rollup(\"count\").last(\"5m\") > 100"

  monitor_thresholds {
    critical = 100
  }

  tags = ["mql-test", "env:test"]
}

# --- Dashboard ---

resource "datadog_dashboard" "test" {
  title       = "MQL Test Dashboard"
  description = "Test dashboard for mql provider verification"
  layout_type = "ordered"

  widget {
    note_definition {
      content          = "This is a test dashboard created by the mql provider Terraform test setup."
      background_color = "white"
      font_size        = "14"
      text_align       = "left"
      show_tick        = false
      tick_edge        = "left"
      tick_pos         = "50%"
    }
  }

  tags = ["mql-test"]
}

# --- Synthetics API Test ---

resource "datadog_synthetics_test" "test_api" {
  name      = "mql-test-api-check"
  type      = "api"
  subtype   = "http"
  status    = "paused"
  message   = "API check failed. @mql-test"
  locations = ["aws:us-east-1"]
  tags      = ["mql-test", "env:test"]

  request_definition {
    method = "GET"
    url    = "https://httpbin.org/status/200"
  }

  assertion {
    type     = "statusCode"
    operator = "is"
    target   = "200"
  }

  options_list {
    tick_every          = 900
    min_failure_duration = 0
    min_location_failed  = 1
  }
}

# --- SLO ---

resource "datadog_service_level_objective" "test" {
  name = "mql-test-slo"
  type = "monitor"

  monitor_ids = [
    datadog_monitor.test_cpu.id,
  ]

  thresholds {
    timeframe = "7d"
    target    = 99.9
    warning   = 99.95
  }

  thresholds {
    timeframe = "30d"
    target    = 99.5
  }

  tags = ["mql-test", "env:test"]
}

# --- Downtime ---

resource "datadog_downtime_schedule" "test" {
  scope = "env:mql-test"

  one_time_schedule {
    start = timeadd(timestamp(), "1h")
    end   = timeadd(timestamp(), "2h")
  }

  display_timezone = "UTC"
  message          = "MQL test downtime - scheduled maintenance window"

  notify_end_states = ["alert", "warn"]
  notify_end_types  = ["canceled", "expired"]
}

# --- Security Monitoring Rule ---

resource "datadog_security_monitoring_rule" "test" {
  name    = "mql-test-security-rule"
  message = "Test security rule for mql provider. @mql-test"
  enabled = false

  query {
    name            = "test_query"
    query           = "source:mql-test"
    aggregation     = "count"
    group_by_fields = []
  }

  case {
    name      = "high"
    status    = "high"
    condition = "test_query > 100"
  }

  options {
    detection_method       = "threshold"
    evaluation_window      = 300
    keep_alive             = 600
    max_signal_duration    = 900
  }

  tags = ["mql-test"]
}

# --- Sensitive Data Scanner ---

resource "datadog_sensitive_data_scanner_group" "test" {
  name        = "mql-test-scanner-group"
  description = "Test scanner group for mql provider verification"
  is_enabled  = false

  filter {
    query = "source:mql-test"
  }

  product_list = ["logs"]
}

# --- Security Monitoring Filter ---

resource "datadog_security_monitoring_filter" "test" {
  name             = "mql-test-security-filter"
  query            = "source:mql-test"
  is_enabled       = false
  filtered_data_type = "logs"

  exclusion_filter {
    name  = "test-exclusion"
    query = "host:excluded"
  }
}

# --- Outputs ---

output "monitor_cpu_id" {
  value = datadog_monitor.test_cpu.id
}

output "monitor_log_id" {
  value = datadog_monitor.test_log.id
}

output "dashboard_id" {
  value = datadog_dashboard.test.id
}

output "synthetics_test_id" {
  value = datadog_synthetics_test.test_api.id
}

output "slo_id" {
  value = datadog_service_level_objective.test.id
}

output "user_id" {
  value = datadog_user.test.id
}

output "role_id" {
  value = datadog_role.test.id
}
