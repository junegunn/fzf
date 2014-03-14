require 'rubygems/dependency_installer'

if Gem::Version.new(RUBY_VERSION) >= Gem::Version.new('2.1.0')
  Gem::DependencyInstaller.new.install 'curses', '~> 1.0'
end

File.open(File.expand_path('../Rakefile', __FILE__), 'w') do |f|
  f.puts 'task :default'
end
