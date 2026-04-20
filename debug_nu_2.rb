require_relative 'test/lib/common'
tmux = Tmux.new(:nushell)
sleep 1
# Simulate exactly what prepare() does, step by step
puts "=== Step 1: send ' ', C-u, Enter ==="
tmux.send_keys ' ', 'C-u', :Enter
sleep 1
lines = tmux.capture
lines.each_with_index { |l, i| puts "#{i}: [#{l}]" }
puts "=== Step 2: send message, Left, Right ==="
tmux.send_keys 'Prepare[0]', :Left, :Right
sleep 1
lines = tmux.capture
lines.each_with_index { |l, i| puts "#{i}: [#{l}]" }
puts "lines[-1] = [#{lines[-1]}]"
puts "match? #{lines[-1] == 'Prepare[0]'}"
puts "=== Step 3: C-u, C-l ==="
tmux.send_keys 'C-u', 'C-l'
sleep 1
lines = tmux.capture
lines.each_with_index { |l, i| puts "#{i}: [#{l}]" }
puts "=== Step 4: second prepare cycle ==="
tmux.send_keys ' ', 'C-u', :Enter, 'Prepare[1]', :Left, :Right
sleep 1
lines = tmux.capture
lines.each_with_index { |l, i| puts "#{i}: [#{l}]" }
puts "lines[-1] = [#{lines[-1]}]"
tmux.kill
