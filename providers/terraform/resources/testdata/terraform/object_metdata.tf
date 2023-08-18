resource "google_compute_instance" "default" {
  name         = "test"
  machine_type = "e2-medium"
  zone         = "us-central1-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-9"
    }
  }

  // Local SSD disk
  scratch_disk {
    interface = "SCSI"
  }

  // metadata is a nested object and no block
  metadata = {
    enable-oslogin = true
  }

  network_interface {
    network = "default"

    access_config {
      // Ephemeral public IP
    }
  }
}
