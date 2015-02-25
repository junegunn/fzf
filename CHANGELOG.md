CHANGELOG
=========

0.9.4
-----

#### New features

- Added `--tac` option to reverse the order of the input.
    - One might argue that this option is unnecessary since we can already put
      `tac` or `tail -r` in the command pipeline to achieve the same result.
      However, the advantage of `--tac` is that it does not block until the
      input is complete.

#### *Backward incompatible changes*

- `--no-sort` option will no longer reverse the display order. You may want to
  use the new `--tac` option with `--no-sort`.
```
history | fzf +s --tac
```

0.9.3
-----

#### New features
- Added `--sync` option for multi-staged filtering

#### Improvements
- `--select-1` and `--exit-0` will start finder immediately when the condition
  cannot be met

