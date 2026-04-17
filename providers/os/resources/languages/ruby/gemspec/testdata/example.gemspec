lib = File.expand_path("lib", __dir__)
$LOAD_PATH.unshift(lib) unless $LOAD_PATH.include?(lib)

Gem::Specification.new do |spec|
  spec.name          = "inspec"
  spec.version       = "6.8.1"
  spec.authors       = ["Chef InSpec Team"]
  spec.email         = ["inspec@chef.io"]
  spec.summary       = "Infrastructure and compliance testing."
  spec.homepage      = "https://github.com/inspec/inspec"
  spec.license       = "Apache-2.0"

  spec.add_dependency "train", "~> 3.10"
  spec.add_dependency "rake"
  spec.add_dependency "mongo", "= 2.13.2"
  spec.add_development_dependency "minitest", "~> 5.0"
end
