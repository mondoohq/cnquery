# Ansible Provider

he Ansible provider enables security and compliance verification of Ansible playbooks using cnquery.

## Get started

```shell
±> cnquery shell ansible providers/ansible/play/testdata/play_cert_validation.yaml
→ connected to Ansible Playbook
  ___ _ __   __ _ _   _  ___ _ __ _   _ 
 / __| '_ \ / _` | | | |/ _ \ '__| | | |
| (__| | | | (_| | |_| |  __/ |  | |_| |
 \___|_| |_|\__, |\__,_|\___|_|   \__, |
  mondoo™      |_|                |___/  interactive shell

cnquery> ansible.plays
ansible.plays: [
  0: ansible.play name="Install packages"
]
```

## Common Queries

Query all plays in a playbook:

```javascript
ansible.plays
```

Access specific play details:

```javascript
ansible.plays.first.name
```

## Example

Assume the following ansible tasks where we install httpd
with [yum](https://docs.ansible.com/projects/ansible/latest/collections/ansible/builtin/dnf_module.html#ansible-collections-ansible-builtin-dnf-module):

```yaml
- name: Install packages
  hosts: all
  gather_facts: false
  tasks:
    - name: Install httpd server
      ansible.builtin.yum:
        name: httpd>=2.4
        state: present
        validate_certs: false
```

You can easily query all tasks for all plays in the playbook:

```shell
cnquery> ansible.plays.map(tasks)
ansible.plays.map: [
  0: [
    0: {
      name: "Install httpd server"
    }
  ]
]
```

You can also query for all tasks that use `ansible.builtin.yum`:

```shell
cnquery> ansible.plays { tasks.where (action["ansible.builtin.yum"] != empty) }
ansible.plays: [
  0: {
    tasks.where: [
      0: ansible.task name="Install httpd server"
    ]
  }
]
```

To enforce that no `ansible.builtin.yum` is using `validate_certs: false`, you write the following MQL:

```shell
ansible.plays.all(
  tasks.where(action["ansible.builtin.yum"] != empty).all(
    action["ansible.builtin.yum"]["validate_certs"] != false 
  )
)
```

Query packs allow you to collect information from your Ansible playbooks without enforcing compliance. Create a query
pack to identify tasks that disable certificate validation:

```yaml
packs:
  - uid: ansible-example-pack
    name: Ansible Example Pack
    version: 1.0.0
    license: BUSL-1.1
    authors:
      - name: Mondoo, Inc
        email: hello@mondoo.com
    groups:
      - title: Query tasks that use insecure yum
        filters: asset.platform == 'ansible-playbook'
        queries:
          - uid: ansible-example-pack-yum-validate-certs
            title: Ansible tasks that do not validate yum certificates
            mql: |
              ansible.plays { 
                tasks.where (action["ansible.builtin.yum"]["validate_certs"] == false )
              }
```

Execute the query pack and format the output with `jq`:

```shell
cnquery scan ansible providers/ansible/play/testdata/play_cert_validation.yaml -f providers/ansible/examples/querypack.mql.yaml --output json | jq .
```

Policies enforce security and compliance standards by defining checks that must pass. Create a policy to ensure
`validate_certs` is always enabled for yum tasks:

```yaml
policies:
  - uid: ansible-example-policy
    name: Ansible Example Policy
    version: 1.0.0
    license: BUSL-1.1
    require:
      - provider: ansible
    authors:
      - name: Mondoo, Inc
        email: hello@mondoo.com
    groups:
      - title: Insecure permissions
        filters: |
          asset.platform == 'ansible-playbook'
        checks:
          - uid: ansible-example-policy-yum-validate-cert
            title: Ensure `validate_certs` is enabled for `ansible.builtin.yum`
            mql: |
              ansible.plays.all(
                tasks.where(action["ansible.builtin.yum"] != empty).all(
                  action["ansible.builtin.yum"]["validate_certs"] != false 
                )
              )
```

Execute the policy scan:

```shell
cnspec scan ansible providers/ansible/play/testdata/play_cert_validation.yaml -f providers/ansible/examples/policy.mql.yaml
```