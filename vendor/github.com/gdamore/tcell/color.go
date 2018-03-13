// Copyright 2015 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tcell

import "strconv"

// Color represents a color.  The low numeric values are the same as used
// by ECMA-48, and beyond that XTerm.  A 24-bit RGB value may be used by
// adding in the ColorIsRGB flag.  For Color names we use the W3C approved
// color names.
//
// Note that on various terminals colors may be approximated however, or
// not supported at all.  If no suitable representation for a color is known,
// the library will simply not set any color, deferring to whatever default
// attributes the terminal uses.
type Color int32

const (
	// ColorDefault is used to leave the Color unchanged from whatever
	// system or teminal default may exist.
	ColorDefault Color = -1

	// ColorIsRGB is used to indicate that the numeric value is not
	// a known color constant, but rather an RGB value.  The lower
	// order 3 bytes are RGB.
	ColorIsRGB Color = 1 << 24
)

// Note that the order of these options is important -- it follows the
// definitions used by ECMA and XTerm.  Hence any further named colors
// must begin at a value not less than 256.
const (
	ColorBlack Color = iota
	ColorMaroon
	ColorGreen
	ColorOlive
	ColorNavy
	ColorPurple
	ColorTeal
	ColorSilver
	ColorGray
	ColorRed
	ColorLime
	ColorYellow
	ColorBlue
	ColorFuchsia
	ColorAqua
	ColorWhite
	Color16
	Color17
	Color18
	Color19
	Color20
	Color21
	Color22
	Color23
	Color24
	Color25
	Color26
	Color27
	Color28
	Color29
	Color30
	Color31
	Color32
	Color33
	Color34
	Color35
	Color36
	Color37
	Color38
	Color39
	Color40
	Color41
	Color42
	Color43
	Color44
	Color45
	Color46
	Color47
	Color48
	Color49
	Color50
	Color51
	Color52
	Color53
	Color54
	Color55
	Color56
	Color57
	Color58
	Color59
	Color60
	Color61
	Color62
	Color63
	Color64
	Color65
	Color66
	Color67
	Color68
	Color69
	Color70
	Color71
	Color72
	Color73
	Color74
	Color75
	Color76
	Color77
	Color78
	Color79
	Color80
	Color81
	Color82
	Color83
	Color84
	Color85
	Color86
	Color87
	Color88
	Color89
	Color90
	Color91
	Color92
	Color93
	Color94
	Color95
	Color96
	Color97
	Color98
	Color99
	Color100
	Color101
	Color102
	Color103
	Color104
	Color105
	Color106
	Color107
	Color108
	Color109
	Color110
	Color111
	Color112
	Color113
	Color114
	Color115
	Color116
	Color117
	Color118
	Color119
	Color120
	Color121
	Color122
	Color123
	Color124
	Color125
	Color126
	Color127
	Color128
	Color129
	Color130
	Color131
	Color132
	Color133
	Color134
	Color135
	Color136
	Color137
	Color138
	Color139
	Color140
	Color141
	Color142
	Color143
	Color144
	Color145
	Color146
	Color147
	Color148
	Color149
	Color150
	Color151
	Color152
	Color153
	Color154
	Color155
	Color156
	Color157
	Color158
	Color159
	Color160
	Color161
	Color162
	Color163
	Color164
	Color165
	Color166
	Color167
	Color168
	Color169
	Color170
	Color171
	Color172
	Color173
	Color174
	Color175
	Color176
	Color177
	Color178
	Color179
	Color180
	Color181
	Color182
	Color183
	Color184
	Color185
	Color186
	Color187
	Color188
	Color189
	Color190
	Color191
	Color192
	Color193
	Color194
	Color195
	Color196
	Color197
	Color198
	Color199
	Color200
	Color201
	Color202
	Color203
	Color204
	Color205
	Color206
	Color207
	Color208
	Color209
	Color210
	Color211
	Color212
	Color213
	Color214
	Color215
	Color216
	Color217
	Color218
	Color219
	Color220
	Color221
	Color222
	Color223
	Color224
	Color225
	Color226
	Color227
	Color228
	Color229
	Color230
	Color231
	Color232
	Color233
	Color234
	Color235
	Color236
	Color237
	Color238
	Color239
	Color240
	Color241
	Color242
	Color243
	Color244
	Color245
	Color246
	Color247
	Color248
	Color249
	Color250
	Color251
	Color252
	Color253
	Color254
	Color255
	ColorAliceBlue
	ColorAntiqueWhite
	ColorAquaMarine
	ColorAzure
	ColorBeige
	ColorBisque
	ColorBlanchedAlmond
	ColorBlueViolet
	ColorBrown
	ColorBurlyWood
	ColorCadetBlue
	ColorChartreuse
	ColorChocolate
	ColorCoral
	ColorCornflowerBlue
	ColorCornsilk
	ColorCrimson
	ColorDarkBlue
	ColorDarkCyan
	ColorDarkGoldenrod
	ColorDarkGray
	ColorDarkGreen
	ColorDarkKhaki
	ColorDarkMagenta
	ColorDarkOliveGreen
	ColorDarkOrange
	ColorDarkOrchid
	ColorDarkRed
	ColorDarkSalmon
	ColorDarkSeaGreen
	ColorDarkSlateBlue
	ColorDarkSlateGray
	ColorDarkTurquoise
	ColorDarkViolet
	ColorDeepPink
	ColorDeepSkyBlue
	ColorDimGray
	ColorDodgerBlue
	ColorFireBrick
	ColorFloralWhite
	ColorForestGreen
	ColorGainsboro
	ColorGhostWhite
	ColorGold
	ColorGoldenrod
	ColorGreenYellow
	ColorHoneydew
	ColorHotPink
	ColorIndianRed
	ColorIndigo
	ColorIvory
	ColorKhaki
	ColorLavender
	ColorLavenderBlush
	ColorLawnGreen
	ColorLemonChiffon
	ColorLightBlue
	ColorLightCoral
	ColorLightCyan
	ColorLightGoldenrodYellow
	ColorLightGray
	ColorLightGreen
	ColorLightPink
	ColorLightSalmon
	ColorLightSeaGreen
	ColorLightSkyBlue
	ColorLightSlateGray
	ColorLightSteelBlue
	ColorLightYellow
	ColorLimeGreen
	ColorLinen
	ColorMediumAquamarine
	ColorMediumBlue
	ColorMediumOrchid
	ColorMediumPurple
	ColorMediumSeaGreen
	ColorMediumSlateBlue
	ColorMediumSpringGreen
	ColorMediumTurquoise
	ColorMediumVioletRed
	ColorMidnightBlue
	ColorMintCream
	ColorMistyRose
	ColorMoccasin
	ColorNavajoWhite
	ColorOldLace
	ColorOliveDrab
	ColorOrange
	ColorOrangeRed
	ColorOrchid
	ColorPaleGoldenrod
	ColorPaleGreen
	ColorPaleTurquoise
	ColorPaleVioletRed
	ColorPapayaWhip
	ColorPeachPuff
	ColorPeru
	ColorPink
	ColorPlum
	ColorPowderBlue
	ColorRebeccaPurple
	ColorRosyBrown
	ColorRoyalBlue
	ColorSaddleBrown
	ColorSalmon
	ColorSandyBrown
	ColorSeaGreen
	ColorSeashell
	ColorSienna
	ColorSkyblue
	ColorSlateBlue
	ColorSlateGray
	ColorSnow
	ColorSpringGreen
	ColorSteelBlue
	ColorTan
	ColorThistle
	ColorTomato
	ColorTurquoise
	ColorViolet
	ColorWheat
	ColorWhiteSmoke
	ColorYellowGreen
)

// These are aliases for the color gray, because some of us spell
// it as grey.
const (
	ColorGrey           = ColorGray
	ColorDimGrey        = ColorDimGray
	ColorDarkGrey       = ColorDarkGray
	ColorDarkSlateGrey  = ColorDarkSlateGray
	ColorLightGrey      = ColorLightGray
	ColorLightSlateGrey = ColorLightSlateGray
	ColorSlateGrey      = ColorSlateGray
)

// ColorValues maps color constants to their RGB values.
var ColorValues = map[Color]int32{
	ColorBlack:                0x000000,
	ColorMaroon:               0x800000,
	ColorGreen:                0x008000,
	ColorOlive:                0x808000,
	ColorNavy:                 0x000080,
	ColorPurple:               0x800080,
	ColorTeal:                 0x008080,
	ColorSilver:               0xC0C0C0,
	ColorGray:                 0x808080,
	ColorRed:                  0xFF0000,
	ColorLime:                 0x00FF00,
	ColorYellow:               0xFFFF00,
	ColorBlue:                 0x0000FF,
	ColorFuchsia:              0xFF00FF,
	ColorAqua:                 0x00FFFF,
	ColorWhite:                0xFFFFFF,
	Color16:                   0x000000, // black
	Color17:                   0x00005F,
	Color18:                   0x000087,
	Color19:                   0x0000AF,
	Color20:                   0x0000D7,
	Color21:                   0x0000FF, // blue
	Color22:                   0x005F00,
	Color23:                   0x005F5F,
	Color24:                   0x005F87,
	Color25:                   0x005FAF,
	Color26:                   0x005FD7,
	Color27:                   0x005FFF,
	Color28:                   0x008700,
	Color29:                   0x00875F,
	Color30:                   0x008787,
	Color31:                   0x0087Af,
	Color32:                   0x0087D7,
	Color33:                   0x0087FF,
	Color34:                   0x00AF00,
	Color35:                   0x00AF5F,
	Color36:                   0x00AF87,
	Color37:                   0x00AFAF,
	Color38:                   0x00AFD7,
	Color39:                   0x00AFFF,
	Color40:                   0x00D700,
	Color41:                   0x00D75F,
	Color42:                   0x00D787,
	Color43:                   0x00D7AF,
	Color44:                   0x00D7D7,
	Color45:                   0x00D7FF,
	Color46:                   0x00FF00, // lime
	Color47:                   0x00FF5F,
	Color48:                   0x00FF87,
	Color49:                   0x00FFAF,
	Color50:                   0x00FFd7,
	Color51:                   0x00FFFF, // aqua
	Color52:                   0x5F0000,
	Color53:                   0x5F005F,
	Color54:                   0x5F0087,
	Color55:                   0x5F00AF,
	Color56:                   0x5F00D7,
	Color57:                   0x5F00FF,
	Color58:                   0x5F5F00,
	Color59:                   0x5F5F5F,
	Color60:                   0x5F5F87,
	Color61:                   0x5F5FAF,
	Color62:                   0x5F5FD7,
	Color63:                   0x5F5FFF,
	Color64:                   0x5F8700,
	Color65:                   0x5F875F,
	Color66:                   0x5F8787,
	Color67:                   0x5F87AF,
	Color68:                   0x5F87D7,
	Color69:                   0x5F87FF,
	Color70:                   0x5FAF00,
	Color71:                   0x5FAF5F,
	Color72:                   0x5FAF87,
	Color73:                   0x5FAFAF,
	Color74:                   0x5FAFD7,
	Color75:                   0x5FAFFF,
	Color76:                   0x5FD700,
	Color77:                   0x5FD75F,
	Color78:                   0x5FD787,
	Color79:                   0x5FD7AF,
	Color80:                   0x5FD7D7,
	Color81:                   0x5FD7FF,
	Color82:                   0x5FFF00,
	Color83:                   0x5FFF5F,
	Color84:                   0x5FFF87,
	Color85:                   0x5FFFAF,
	Color86:                   0x5FFFD7,
	Color87:                   0x5FFFFF,
	Color88:                   0x870000,
	Color89:                   0x87005F,
	Color90:                   0x870087,
	Color91:                   0x8700AF,
	Color92:                   0x8700D7,
	Color93:                   0x8700FF,
	Color94:                   0x875F00,
	Color95:                   0x875F5F,
	Color96:                   0x875F87,
	Color97:                   0x875FAF,
	Color98:                   0x875FD7,
	Color99:                   0x875FFF,
	Color100:                  0x878700,
	Color101:                  0x87875F,
	Color102:                  0x878787,
	Color103:                  0x8787AF,
	Color104:                  0x8787D7,
	Color105:                  0x8787FF,
	Color106:                  0x87AF00,
	Color107:                  0x87AF5F,
	Color108:                  0x87AF87,
	Color109:                  0x87AFAF,
	Color110:                  0x87AFD7,
	Color111:                  0x87AFFF,
	Color112:                  0x87D700,
	Color113:                  0x87D75F,
	Color114:                  0x87D787,
	Color115:                  0x87D7AF,
	Color116:                  0x87D7D7,
	Color117:                  0x87D7FF,
	Color118:                  0x87FF00,
	Color119:                  0x87FF5F,
	Color120:                  0x87FF87,
	Color121:                  0x87FFAF,
	Color122:                  0x87FFD7,
	Color123:                  0x87FFFF,
	Color124:                  0xAF0000,
	Color125:                  0xAF005F,
	Color126:                  0xAF0087,
	Color127:                  0xAF00AF,
	Color128:                  0xAF00D7,
	Color129:                  0xAF00FF,
	Color130:                  0xAF5F00,
	Color131:                  0xAF5F5F,
	Color132:                  0xAF5F87,
	Color133:                  0xAF5FAF,
	Color134:                  0xAF5FD7,
	Color135:                  0xAF5FFF,
	Color136:                  0xAF8700,
	Color137:                  0xAF875F,
	Color138:                  0xAF8787,
	Color139:                  0xAF87AF,
	Color140:                  0xAF87D7,
	Color141:                  0xAF87FF,
	Color142:                  0xAFAF00,
	Color143:                  0xAFAF5F,
	Color144:                  0xAFAF87,
	Color145:                  0xAFAFAF,
	Color146:                  0xAFAFD7,
	Color147:                  0xAFAFFF,
	Color148:                  0xAFD700,
	Color149:                  0xAFD75F,
	Color150:                  0xAFD787,
	Color151:                  0xAFD7AF,
	Color152:                  0xAFD7D7,
	Color153:                  0xAFD7FF,
	Color154:                  0xAFFF00,
	Color155:                  0xAFFF5F,
	Color156:                  0xAFFF87,
	Color157:                  0xAFFFAF,
	Color158:                  0xAFFFD7,
	Color159:                  0xAFFFFF,
	Color160:                  0xD70000,
	Color161:                  0xD7005F,
	Color162:                  0xD70087,
	Color163:                  0xD700AF,
	Color164:                  0xD700D7,
	Color165:                  0xD700FF,
	Color166:                  0xD75F00,
	Color167:                  0xD75F5F,
	Color168:                  0xD75F87,
	Color169:                  0xD75FAF,
	Color170:                  0xD75FD7,
	Color171:                  0xD75FFF,
	Color172:                  0xD78700,
	Color173:                  0xD7875F,
	Color174:                  0xD78787,
	Color175:                  0xD787AF,
	Color176:                  0xD787D7,
	Color177:                  0xD787FF,
	Color178:                  0xD7AF00,
	Color179:                  0xD7AF5F,
	Color180:                  0xD7AF87,
	Color181:                  0xD7AFAF,
	Color182:                  0xD7AFD7,
	Color183:                  0xD7AFFF,
	Color184:                  0xD7D700,
	Color185:                  0xD7D75F,
	Color186:                  0xD7D787,
	Color187:                  0xD7D7AF,
	Color188:                  0xD7D7D7,
	Color189:                  0xD7D7FF,
	Color190:                  0xD7FF00,
	Color191:                  0xD7FF5F,
	Color192:                  0xD7FF87,
	Color193:                  0xD7FFAF,
	Color194:                  0xD7FFD7,
	Color195:                  0xD7FFFF,
	Color196:                  0xFF0000, // red
	Color197:                  0xFF005F,
	Color198:                  0xFF0087,
	Color199:                  0xFF00AF,
	Color200:                  0xFF00D7,
	Color201:                  0xFF00FF, // fuchsia
	Color202:                  0xFF5F00,
	Color203:                  0xFF5F5F,
	Color204:                  0xFF5F87,
	Color205:                  0xFF5FAF,
	Color206:                  0xFF5FD7,
	Color207:                  0xFF5FFF,
	Color208:                  0xFF8700,
	Color209:                  0xFF875F,
	Color210:                  0xFF8787,
	Color211:                  0xFF87AF,
	Color212:                  0xFF87D7,
	Color213:                  0xFF87FF,
	Color214:                  0xFFAF00,
	Color215:                  0xFFAF5F,
	Color216:                  0xFFAF87,
	Color217:                  0xFFAFAF,
	Color218:                  0xFFAFD7,
	Color219:                  0xFFAFFF,
	Color220:                  0xFFD700,
	Color221:                  0xFFD75F,
	Color222:                  0xFFD787,
	Color223:                  0xFFD7AF,
	Color224:                  0xFFD7D7,
	Color225:                  0xFFD7FF,
	Color226:                  0xFFFF00, // yellow
	Color227:                  0xFFFF5F,
	Color228:                  0xFFFF87,
	Color229:                  0xFFFFAF,
	Color230:                  0xFFFFD7,
	Color231:                  0xFFFFFF, // white
	Color232:                  0x080808,
	Color233:                  0x121212,
	Color234:                  0x1C1C1C,
	Color235:                  0x262626,
	Color236:                  0x303030,
	Color237:                  0x3A3A3A,
	Color238:                  0x444444,
	Color239:                  0x4E4E4E,
	Color240:                  0x585858,
	Color241:                  0x626262,
	Color242:                  0x6C6C6C,
	Color243:                  0x767676,
	Color244:                  0x808080, // grey
	Color245:                  0x8A8A8A,
	Color246:                  0x949494,
	Color247:                  0x9E9E9E,
	Color248:                  0xA8A8A8,
	Color249:                  0xB2B2B2,
	Color250:                  0xBCBCBC,
	Color251:                  0xC6C6C6,
	Color252:                  0xD0D0D0,
	Color253:                  0xDADADA,
	Color254:                  0xE4E4E4,
	Color255:                  0xEEEEEE,
	ColorAliceBlue:            0xF0F8FF,
	ColorAntiqueWhite:         0xFAEBD7,
	ColorAquaMarine:           0x7FFFD4,
	ColorAzure:                0xF0FFFF,
	ColorBeige:                0xF5F5DC,
	ColorBisque:               0xFFE4C4,
	ColorBlanchedAlmond:       0xFFEBCD,
	ColorBlueViolet:           0x8A2BE2,
	ColorBrown:                0xA52A2A,
	ColorBurlyWood:            0xDEB887,
	ColorCadetBlue:            0x5F9EA0,
	ColorChartreuse:           0x7FFF00,
	ColorChocolate:            0xD2691E,
	ColorCoral:                0xFF7F50,
	ColorCornflowerBlue:       0x6495ED,
	ColorCornsilk:             0xFFF8DC,
	ColorCrimson:              0xDC143C,
	ColorDarkBlue:             0x00008B,
	ColorDarkCyan:             0x008B8B,
	ColorDarkGoldenrod:        0xB8860B,
	ColorDarkGray:             0xA9A9A9,
	ColorDarkGreen:            0x006400,
	ColorDarkKhaki:            0xBDB76B,
	ColorDarkMagenta:          0x8B008B,
	ColorDarkOliveGreen:       0x556B2F,
	ColorDarkOrange:           0xFF8C00,
	ColorDarkOrchid:           0x9932CC,
	ColorDarkRed:              0x8B0000,
	ColorDarkSalmon:           0xE9967A,
	ColorDarkSeaGreen:         0x8FBC8F,
	ColorDarkSlateBlue:        0x483D8B,
	ColorDarkSlateGray:        0x2F4F4F,
	ColorDarkTurquoise:        0x00CED1,
	ColorDarkViolet:           0x9400D3,
	ColorDeepPink:             0xFF1493,
	ColorDeepSkyBlue:          0x00BFFF,
	ColorDimGray:              0x696969,
	ColorDodgerBlue:           0x1E90FF,
	ColorFireBrick:            0xB22222,
	ColorFloralWhite:          0xFFFAF0,
	ColorForestGreen:          0x228B22,
	ColorGainsboro:            0xDCDCDC,
	ColorGhostWhite:           0xF8F8FF,
	ColorGold:                 0xFFD700,
	ColorGoldenrod:            0xDAA520,
	ColorGreenYellow:          0xADFF2F,
	ColorHoneydew:             0xF0FFF0,
	ColorHotPink:              0xFF69B4,
	ColorIndianRed:            0xCD5C5C,
	ColorIndigo:               0x4B0082,
	ColorIvory:                0xFFFFF0,
	ColorKhaki:                0xF0E68C,
	ColorLavender:             0xE6E6FA,
	ColorLavenderBlush:        0xFFF0F5,
	ColorLawnGreen:            0x7CFC00,
	ColorLemonChiffon:         0xFFFACD,
	ColorLightBlue:            0xADD8E6,
	ColorLightCoral:           0xF08080,
	ColorLightCyan:            0xE0FFFF,
	ColorLightGoldenrodYellow: 0xFAFAD2,
	ColorLightGray:            0xD3D3D3,
	ColorLightGreen:           0x90EE90,
	ColorLightPink:            0xFFB6C1,
	ColorLightSalmon:          0xFFA07A,
	ColorLightSeaGreen:        0x20B2AA,
	ColorLightSkyBlue:         0x87CEFA,
	ColorLightSlateGray:       0x778899,
	ColorLightSteelBlue:       0xB0C4DE,
	ColorLightYellow:          0xFFFFE0,
	ColorLimeGreen:            0x32CD32,
	ColorLinen:                0xFAF0E6,
	ColorMediumAquamarine:     0x66CDAA,
	ColorMediumBlue:           0x0000CD,
	ColorMediumOrchid:         0xBA55D3,
	ColorMediumPurple:         0x9370DB,
	ColorMediumSeaGreen:       0x3CB371,
	ColorMediumSlateBlue:      0x7B68EE,
	ColorMediumSpringGreen:    0x00FA9A,
	ColorMediumTurquoise:      0x48D1CC,
	ColorMediumVioletRed:      0xC71585,
	ColorMidnightBlue:         0x191970,
	ColorMintCream:            0xF5FFFA,
	ColorMistyRose:            0xFFE4E1,
	ColorMoccasin:             0xFFE4B5,
	ColorNavajoWhite:          0xFFDEAD,
	ColorOldLace:              0xFDF5E6,
	ColorOliveDrab:            0x6B8E23,
	ColorOrange:               0xFFA500,
	ColorOrangeRed:            0xFF4500,
	ColorOrchid:               0xDA70D6,
	ColorPaleGoldenrod:        0xEEE8AA,
	ColorPaleGreen:            0x98FB98,
	ColorPaleTurquoise:        0xAFEEEE,
	ColorPaleVioletRed:        0xDB7093,
	ColorPapayaWhip:           0xFFEFD5,
	ColorPeachPuff:            0xFFDAB9,
	ColorPeru:                 0xCD853F,
	ColorPink:                 0xFFC0CB,
	ColorPlum:                 0xDDA0DD,
	ColorPowderBlue:           0xB0E0E6,
	ColorRebeccaPurple:        0x663399,
	ColorRosyBrown:            0xBC8F8F,
	ColorRoyalBlue:            0x4169E1,
	ColorSaddleBrown:          0x8B4513,
	ColorSalmon:               0xFA8072,
	ColorSandyBrown:           0xF4A460,
	ColorSeaGreen:             0x2E8B57,
	ColorSeashell:             0xFFF5EE,
	ColorSienna:               0xA0522D,
	ColorSkyblue:              0x87CEEB,
	ColorSlateBlue:            0x6A5ACD,
	ColorSlateGray:            0x708090,
	ColorSnow:                 0xFFFAFA,
	ColorSpringGreen:          0x00FF7F,
	ColorSteelBlue:            0x4682B4,
	ColorTan:                  0xD2B48C,
	ColorThistle:              0xD8BFD8,
	ColorTomato:               0xFF6347,
	ColorTurquoise:            0x40E0D0,
	ColorViolet:               0xEE82EE,
	ColorWheat:                0xF5DEB3,
	ColorWhiteSmoke:           0xF5F5F5,
	ColorYellowGreen:          0x9ACD32,
}

// ColorNames holds the written names of colors. Useful to present a list of
// recognized named colors.
var ColorNames = map[string]Color{
	"black":                ColorBlack,
	"maroon":               ColorMaroon,
	"green":                ColorGreen,
	"olive":                ColorOlive,
	"navy":                 ColorNavy,
	"purple":               ColorPurple,
	"teal":                 ColorTeal,
	"silver":               ColorSilver,
	"gray":                 ColorGray,
	"red":                  ColorRed,
	"lime":                 ColorLime,
	"yellow":               ColorYellow,
	"blue":                 ColorBlue,
	"fuchsia":              ColorFuchsia,
	"aqua":                 ColorAqua,
	"white":                ColorWhite,
	"aliceblue":            ColorAliceBlue,
	"antiquewhite":         ColorAntiqueWhite,
	"aquamarine":           ColorAquaMarine,
	"azure":                ColorAzure,
	"beige":                ColorBeige,
	"bisque":               ColorBisque,
	"blanchedalmond":       ColorBlanchedAlmond,
	"blueviolet":           ColorBlueViolet,
	"brown":                ColorBrown,
	"burlywood":            ColorBurlyWood,
	"cadetblue":            ColorCadetBlue,
	"chartreuse":           ColorChartreuse,
	"chocolate":            ColorChocolate,
	"coral":                ColorCoral,
	"cornflowerblue":       ColorCornflowerBlue,
	"cornsilk":             ColorCornsilk,
	"crimson":              ColorCrimson,
	"darkblue":             ColorDarkBlue,
	"darkcyan":             ColorDarkCyan,
	"darkgoldenrod":        ColorDarkGoldenrod,
	"darkgray":             ColorDarkGray,
	"darkgreen":            ColorDarkGreen,
	"darkkhaki":            ColorDarkKhaki,
	"darkmagenta":          ColorDarkMagenta,
	"darkolivegreen":       ColorDarkOliveGreen,
	"darkorange":           ColorDarkOrange,
	"darkorchid":           ColorDarkOrchid,
	"darkred":              ColorDarkRed,
	"darksalmon":           ColorDarkSalmon,
	"darkseagreen":         ColorDarkSeaGreen,
	"darkslateblue":        ColorDarkSlateBlue,
	"darkslategray":        ColorDarkSlateGray,
	"darkturquoise":        ColorDarkTurquoise,
	"darkviolet":           ColorDarkViolet,
	"deeppink":             ColorDeepPink,
	"deepskyblue":          ColorDeepSkyBlue,
	"dimgray":              ColorDimGray,
	"dodgerblue":           ColorDodgerBlue,
	"firebrick":            ColorFireBrick,
	"floralwhite":          ColorFloralWhite,
	"forestgreen":          ColorForestGreen,
	"gainsboro":            ColorGainsboro,
	"ghostwhite":           ColorGhostWhite,
	"gold":                 ColorGold,
	"goldenrod":            ColorGoldenrod,
	"greenyellow":          ColorGreenYellow,
	"honeydew":             ColorHoneydew,
	"hotpink":              ColorHotPink,
	"indianred":            ColorIndianRed,
	"indigo":               ColorIndigo,
	"ivory":                ColorIvory,
	"khaki":                ColorKhaki,
	"lavender":             ColorLavender,
	"lavenderblush":        ColorLavenderBlush,
	"lawngreen":            ColorLawnGreen,
	"lemonchiffon":         ColorLemonChiffon,
	"lightblue":            ColorLightBlue,
	"lightcoral":           ColorLightCoral,
	"lightcyan":            ColorLightCyan,
	"lightgoldenrodyellow": ColorLightGoldenrodYellow,
	"lightgray":            ColorLightGray,
	"lightgreen":           ColorLightGreen,
	"lightpink":            ColorLightPink,
	"lightsalmon":          ColorLightSalmon,
	"lightseagreen":        ColorLightSeaGreen,
	"lightskyblue":         ColorLightSkyBlue,
	"lightslategray":       ColorLightSlateGray,
	"lightsteelblue":       ColorLightSteelBlue,
	"lightyellow":          ColorLightYellow,
	"limegreen":            ColorLimeGreen,
	"linen":                ColorLinen,
	"mediumaquamarine":     ColorMediumAquamarine,
	"mediumblue":           ColorMediumBlue,
	"mediumorchid":         ColorMediumOrchid,
	"mediumpurple":         ColorMediumPurple,
	"mediumseagreen":       ColorMediumSeaGreen,
	"mediumslateblue":      ColorMediumSlateBlue,
	"mediumspringgreen":    ColorMediumSpringGreen,
	"mediumturquoise":      ColorMediumTurquoise,
	"mediumvioletred":      ColorMediumVioletRed,
	"midnightblue":         ColorMidnightBlue,
	"mintcream":            ColorMintCream,
	"mistyrose":            ColorMistyRose,
	"moccasin":             ColorMoccasin,
	"navajowhite":          ColorNavajoWhite,
	"oldlace":              ColorOldLace,
	"olivedrab":            ColorOliveDrab,
	"orange":               ColorOrange,
	"orangered":            ColorOrangeRed,
	"orchid":               ColorOrchid,
	"palegoldenrod":        ColorPaleGoldenrod,
	"palegreen":            ColorPaleGreen,
	"paleturquoise":        ColorPaleTurquoise,
	"palevioletred":        ColorPaleVioletRed,
	"papayawhip":           ColorPapayaWhip,
	"peachpuff":            ColorPeachPuff,
	"peru":                 ColorPeru,
	"pink":                 ColorPink,
	"plum":                 ColorPlum,
	"powderblue":           ColorPowderBlue,
	"rebeccapurple":        ColorRebeccaPurple,
	"rosybrown":            ColorRosyBrown,
	"royalblue":            ColorRoyalBlue,
	"saddlebrown":          ColorSaddleBrown,
	"salmon":               ColorSalmon,
	"sandybrown":           ColorSandyBrown,
	"seagreen":             ColorSeaGreen,
	"seashell":             ColorSeashell,
	"sienna":               ColorSienna,
	"skyblue":              ColorSkyblue,
	"slateblue":            ColorSlateBlue,
	"slategray":            ColorSlateGray,
	"snow":                 ColorSnow,
	"springgreen":          ColorSpringGreen,
	"steelblue":            ColorSteelBlue,
	"tan":                  ColorTan,
	"thistle":              ColorThistle,
	"tomato":               ColorTomato,
	"turquoise":            ColorTurquoise,
	"violet":               ColorViolet,
	"wheat":                ColorWheat,
	"whitesmoke":           ColorWhiteSmoke,
	"yellowgreen":          ColorYellowGreen,
	"grey":                 ColorGray,
	"dimgrey":              ColorDimGray,
	"darkgrey":             ColorDarkGray,
	"darkslategrey":        ColorDarkSlateGray,
	"lightgrey":            ColorLightGray,
	"lightslategrey":       ColorLightSlateGray,
	"slategrey":            ColorSlateGray,
}

// Hex returns the color's hexadecimal RGB 24-bit value with each component
// consisting of a single byte, ala R << 16 | G << 8 | B.  If the color
// is unknown or unset, -1 is returned.
func (c Color) Hex() int32 {
	if c&ColorIsRGB != 0 {
		return (int32(c) & 0xffffff)
	}
	if v, ok := ColorValues[c]; ok {
		return v
	}
	return -1
}

// RGB returns the red, green, and blue components of the color, with
// each component represented as a value 0-255.  In the event that the
// color cannot be broken up (not set usually), -1 is returned for each value.
func (c Color) RGB() (int32, int32, int32) {
	v := c.Hex()
	if v < 0 {
		return -1, -1, -1
	}
	return (v >> 16) & 0xff, (v >> 8) & 0xff, v & 0xff
}

// NewRGBColor returns a new color with the given red, green, and blue values.
// Each value must be represented in the range 0-255.
func NewRGBColor(r, g, b int32) Color {
	return NewHexColor(((r & 0xff) << 16) | ((g & 0xff) << 8) | (b & 0xff))
}

// NewHexColor returns a color using the given 24-bit RGB value.
func NewHexColor(v int32) Color {
	return ColorIsRGB | Color(v)
}

// GetColor creates a Color from a color name (W3C name). A hex value may
// be supplied as a string in the format "#ffffff".
func GetColor(name string) Color {
	if c, ok := ColorNames[name]; ok {
		return c
	}
	if len(name) == 7 && name[0] == '#' {
		if v, e := strconv.ParseInt(name[1:], 16, 32); e == nil {
			return NewHexColor(int32(v))
		}
	}
	return ColorDefault
}
