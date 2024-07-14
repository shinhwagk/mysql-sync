provider "consul" {
  address    = "http://192.168.2.251:8500"
  datacenter = "dc1"
}

locals {
  # environments = {
  #   "testing" = "http://192.168.199.45:4646"
  # }
  # jobs_path = join("/", [path.root, "jobs-${terraform.workspace}"])
  jobs_path = join("/", [path.root, "mysqlsync-tasks"])
}

locals {
  yaml_files = fileset(local.jobs_path, "*.yml")
  configs = {
    for file in local.yaml_files :
    trimsuffix(file, ".yml") => yamldecode(file("${local.jobs_path}/${file}"))
  }
}

resource "consul_keys" "config" {
  for_each   = local.configs
  datacenter = "dc1"
  key {
    path  = "mysqlsync/${each.value.replication.name}/config.yml"
    value = yamlencode(each.value)
  }
}

resource "local_file" "rendered_files" {
  for_each = local.configs
  filename = "./nomad-jobs/${each.key}.hcl"
  content = templatefile("./templates/nomad.tpl", {
    repl      = each.value.replication
    dests     = each.value.destination.destinations
    dest_keys = keys(each.value.destination.destinations)
    all_datacenters = distinct(concat(
      each.value.replication.nomad.datacenters,
      [for dest in keys(each.value.destination.destinations) : each.value.destination.destinations[dest].nomad.datacenters]...
    ))
    config_key = "mysqlsync/${each.value.replication.name}/config.yml"
  })
}



# resource "nomad_job" "example" {
#   jobspec = templatefile("${path.module}/nomad_job.tpl", {
#     job_name  = local.job_config.job_name
#     region    = local.job_config.region
#     datacenter = local.job_config.datacenter
#     # groups    = local.groups
#   })
# }


# provider "nomad" {
#   address = var.nomad_address == "" ? local.environments[terraform.workspace] : var.nomad_address
# }

# resource "nomad_job" "launch_job" {
#   for_each = fileset(local.jobs_path, "*.hcl")
#   jobspec  = file(join("/", [local.jobs_path, each.value])) // "${jobs_path}/${each.value}")
# }
