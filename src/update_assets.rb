#!/usr/bin/env ruby
# frozen_string_literal: true

# http://www.rubydoc.info/github/rest-client/rest-client/RestClient
require('rest_client')
require('json')

if ARGV.length < 3
  puts("usage: #{$PROGRAM_NAME} <token> <version> <files...>")
  exit(1)
end

token, version, *files = ARGV
base = 'https://api.github.com/repos/junegunn/fzf-bin/releases'

# List releases
rels = JSON.parse(RestClient.get(base, authorization: "token #{token}"))
rel = rels.find { |r| r['tag_name'] == version }
unless rel
  puts("#{version} not found")
  exit(1)
end

# List assets
assets = Hash[rel['assets'].map { |a| a.values_at('name', 'id') }]

files.select { |f| File.exist?(f) }.map do |file|
  Thread.new do
    name = File.basename(file)

    if asset_id = assets[name] # rubocop:todo Lint/AssignmentInCondition
      puts("#{name} found. Deleting asset id #{asset_id}.")
      RestClient.delete("#{base}/assets/#{asset_id}",
                        authorization: "token #{token}")
    else
      puts("#{name} not found")
    end

    puts "Uploading #{name}"
    RestClient.post(
      "#{base.sub('api', 'uploads')}/#{rel['id']}/assets?name=#{name}",
      File.read(file),
      authorization: "token #{token}",
      content_type: 'application/octet-stream'
    )
  end
end.each(&:join)
