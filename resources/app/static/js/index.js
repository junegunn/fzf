$(function() {
  $(".form-control").change(function() {
    updateCode();
    updateView();
  });
});

// Update the 'Generated Code'
function updateCode() {
  $('.code').text("export FZF_DEFAULT_OPTS=$FZF_DEFAULT_OPTS'\n" + getColorOptions() + "'");
  $('.code').html($('.code').html().replace(/\n/g, '<br/>'));
};

// Update the 'Preview'
function updateView() {
  updateColorView($(".fg"), "color", $('#fg').val());
  updateColorView($(".bg"), "background-color", $('#bg').val());
  $(".fgp").css("color", $('#fgp').val());
  $(".bgp").css("background-color", $('#bgp').val());
  $(".hl").css("color", $('#hl').val());
  $(".hlp").css("color", $('#hlp').val());
  $(".inf").css("color", $('#info').val());
  $(".pro").css("color", $('#promp').val());
  $(".poi").css("color", $('#pointer').val());
  $(".mar").css("color", $('#marker').val());
  $(".spi").css("color", $('#spinner').val());
  $(".hea").css("color", $('#header').val());
}

// function to update preview of 'fg' and 'bg' except for -1
function updateColorView(ele, styleType, val) {
  if (val) {
    ele.css(styleType, val);
  }
}

// onclick function of 'default' button
function setDefault(ele) {
  $(ele).val("")
  updateCode();
}

// onclick function of 'reset' button
function resetToDefault() {
  $('.opt').each(function() {
    console.log($(this).minicolors('value', this.defaultValue));
  });
  updateCode();
  updateView();

  $("#reset-btn").popover('show');
  setTimeout(function() {
    $("#reset-btn").popover('destroy');
  }, 1000);
}

// onclick function of 'apply to fzf' button
function applyToFzf() {
  let message = {"name": "sendCode"};
  message.payload = getColorOptions();
  astilectron.sendMessage(message, function(message) {
    // Check error
    if (message.name === "error") {
      asticode.notifier.error(message.payload);
      return
    }
  })
  
  $("#apply-btn").popover('show');
  setTimeout(function() {
    $("#apply-btn").popover('destroy');
  }, 1000);
}

// Make FZF_DEFAULT_OPTS string
function getColorOptions() {
  return "--color=fg:" + getColorText($('#fg')) + ",bg:" + getColorText($('#bg')) + ",hl:" + $('#hl').val() +
  "\n--color=fg+:" + $('#fgp').val() + ",bg+:" + $('#bgp').val() + ",hl+:" + $('#hlp').val() +
  "\n--color=info:" + $('#info').val() + ",prompt:" + $('#promp').val() + ",pointer:" + $('#pointer').val() +
  "\n--color=marker:" + $('#marker').val() + ",spinner:" + $('#spinner').val() + ",header:" + $('#header').val()
}

// function to get a string containing 'fg' and 'bg' = -1
function getColorText(ele) {
  if (ele.val()) {
    return ele.val();
  }
  return -1;
}