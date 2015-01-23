#!/usr/bin/env ruby

# http://www.rubydoc.info/github/rest-client/rest-client/RestClient
require 'rest_client'

if ARGV.length < 3
  puts "usage: #$0 <token> <version> <files...>"
  exit 1
end

token, version, *files = ARGV
base = "https://api.github.com/repos/junegunn/fzf-bin/releases"

# List releases
rels = JSON.parse(RestClient.get(base, :authorization => "token #{token}"))
rel = rels.find { |r| r['tag_name'] == version }
unless rel
  puts "#{version} not found"
  exit 1
end

# List assets
assets = Hash[rel['assets'].map { |a| a.values_at *%w[name id] }]

files.select { |f| File.exists? f }.each do |file|
  name = File.basename file

  if asset_id = assets[name]
    puts "#{name} found. Deleting asset id #{asset_id}."
    RestClient.delete "#{base}/assets/#{asset_id}",
      :authorization => "token #{token}"
  else
    puts "#{name} not found"
  end

  puts "Uploading #{name}"
  RestClient.post(
    "#{base.sub 'api', 'uploads'}/#{rel['id']}/assets?name=#{name}",
    File.read(file),
    :authorization => "token #{token}",
    :content_type  => "application/octet-stream")
end
