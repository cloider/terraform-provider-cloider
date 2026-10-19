variable "name" {
  description = "The app's name, used for the cluster, the database and the Kubernetes objects."
  type        = string
  default     = "shop"
}

variable "registry_project" {
  description = "The registry project's name, which has to be unique across every Cloider account."
  type        = string
}

variable "subdomain" {
  description = "The free subdomain the app is served at, such as acme-shop for acme-shop.cloider.app."
  type        = string
}

variable "image_tag" {
  description = "The tag of the app's image to run, pushed to <registry_url>/<name>:<tag>."
  type        = string
  default     = "latest"
}

variable "replicas" {
  type    = number
  default = 2
}
