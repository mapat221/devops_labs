terraform {
  required_providers {
    yandex = {
      source = "yandex-cloud/yandex"
    }
  }
}

provider "yandex" {
  token     = "token"
  cloud_id  = "cloud_id"
  folder_id = cloud_id"
  zone      = "ru-central1-a"
}

# Сеть
resource "yandex_vpc_network" "lab-network" {
  name = "lab-network"
}

# Подсеть
resource "yandex_vpc_subnet" "lab-subnet" {
  name           = "lab-subnet"
  zone           = "ru-central1-a"
  network_id     = yandex_vpc_network.lab-network.id
  v4_cidr_blocks = ["192.168.10.0/24"]
}

# Находим свежую Ubuntu
data "yandex_compute_image" "ubuntu" {
  family = "ubuntu-2204-lts"
}

# Виртуальная машина
resource "yandex_compute_instance" "minio-server" {
  name        = "minio-server"
  platform_id = "standard-v3"
  zone        = "ru-central1-a"

  resources {
    cores  = 2
    memory = 2
  }

  boot_disk {
    initialize_params {
      image_id = data.yandex_compute_image.ubuntu.id
      size     = 10
    }
  }

  network_interface {
    subnet_id = yandex_vpc_subnet.lab-subnet.id
    nat       = true
  }

  metadata = {
    ssh-keys = "ubuntu:${file("~/.ssh/id_rsa.pub")}"
  }
}

# Выводим IP на экран
output "external_ip" {
  value = yandex_compute_instance.minio-server.network_interface.0.nat_ip_address
}
