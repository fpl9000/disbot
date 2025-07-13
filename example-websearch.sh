#!/bin/bash

curl https://api.anthropic.com/v1/messages \
    --show-headers \
    --header "x-api-key: $ANTHROPIC_API_KEY" \
    --header "anthropic-version: 2023-06-01" \
    --header "content-type: application/json" \
    --data '{
        "model": "claude-sonnet-4-0",
        "max_tokens": 2048,
        "messages": [
            {
                "role": "user",
                "content": "How fast is extraterrestrial comet 3I/ATLAS moving through the solar system?"
            }
        ],
        "tools": [{
            "type": "web_search_20250305",
            "name": "web_search",
            "max_uses": 3
        }]
    }'

exit 0

# Example JSON response:
# 
# HTTP/2 200 
# date: Mon, 07 Jul 2025 16:38:57 GMT
# content-type: application/json
# content-length: 145306
# request-id: req_011CQszBhZxds64EmaBwTZNM
# strict-transport-security: max-age=31536000; includeSubDomains; preload
# anthropic-organization-id: db7657c3-9aec-4116-96b6-f5b7eca57baf
# via: 1.1 google
# cf-cache-status: DYNAMIC
# x-robots-tag: none
# server: cloudflare
# cf-ray: 95b8c5355fa34cfc-BOS
# 
# {"id":"msg_01GQBfzdLww9ED4boBCsB756","type":"message","role":"assistant","model":"claude-opus-4-20250514","content":[
# {"type":"text","text":"I'll search for information about the speed of interstellar comet C/2019 Q4 (Borisov), also known as 2I/Borisov. Note that you mentioned \"3I/ATLAS\" but I believe you're asking about an interstellar object - the second confirmed interstellar object was 2I/Borisov, not 3I/ATLAS."},
# {"type":"server_tool_use","id":"srvtoolu_01JXWKuD7uz5uQqJm3EQ2rQ1","name":"web_search","input":{"query":"2I/Borisov comet speed velocity"}},
# {"type":"web_search_tool_result","tool_use_id":"srvtoolu_01JXWKuD7uz5uQqJm3EQ2rQ1","content":[{"type":"web_search_result","title":"2I/Borisov - Wikipedia","url":"https://en.wikipedia.org/wiki/2I/Borisov","encrypted_content":"...","page_age":"2 days ago"},
# {"type":"web_search_result","title":"Comet 2I/Borisov - NASA Science","url":"https://science.nasa.gov/solar-system/comets/2i-borisov/","encrypted_content":"...","page_age":"November 3, 2024"},
# {"type":"web_search_result","title":"A search for the origin of the interstellar comet 2I/Borisov | Astronomy & Astrophysics (A&A)","url":"https://www.aanda.org/articles/aa/full_html/2020/02/aa37231-19/aa37231-19.html","encrypted_content":"...","page_age":null},
# {"type":"web_search_result","title":"Comet 2I/ Borisov Near Perihelion - NASA Science","url":"https://science.nasa.gov/asset/hubble/comet-2i-borisov-near-perihelion/","encrypted_content":"...","page_age":"March 28, 2025"},
# {"type":"web_search_result","title":"Unusual polarimetric properties for interstellar comet 2I/Borisov | Nature Communications","url":"https://www.nature.com/articles/s41467-021-22000-x","encrypted_content":"...","page_age":null},
# {"type":"web_search_result","title":"Interstellar Comet 2I/Borisov Swings Past Sun - NASA Science","url":"https://science.nasa.gov/missions/hubble/interstellar-comet-2iborisov-swings-past-sun/","encrypted_content":"...","page_age":"March 31, 2025"},
# {"type":"web_search_result","title":"Sending a spacecraft to interstellar comet 2I/Borisov - ScienceDirect","url":"https://www.sciencedirect.com/science/article/abs/pii/S0094576521004926","encrypted_content":"...","page_age":null},
# {"type":"web_search_result","title":"Comet 2I/ Borisov at Perihelion | HubbleSite","url":"https://hubblesite.org/contents/media/images/2019/61/4592-Image?Tag=Small+Solar+System+Bodies","encrypted_content":"...","page_age":null},
# {"type":"web_search_result","title":"Cometary observations in light-polluted environments: a case study of interstellar comet 2I/Borisov | Astrophysics and Space Science","url":"https://link.springer.com/article/10.1007/s10509-025-04424-9","encrypted_content":"...","page_age":null},
# {"type":"web_search_result","title":"Comet 2I/ Borisov at Perihelion - NASA Science","url":"https://science.nasa.gov/asset/hubble/comet-2i-borisov-at-perihelion/","encrypted_content":"...","page_age":"March 28, 2025"}]},
# {"type":"text","text":"Let me search for more specific information about the current speed of any interstellar object, including checking if there's a 3I object that has been discovered."},
# {"type":"server_tool_use","id":"srvtoolu_01XR4YyTnmG5hanW4fNQ7VAu","name":"web_search","input":{"query":"3I/ATLAS interstellar object 2024"}},
# {"type":"web_search_tool_result","tool_use_id":"srvtoolu_01XR4YyTnmG5hanW4fNQ7VAu","content":[{"type":"web_search_result","title":"Mysterious Object Headed Into Our Solar System Is Coming From the Center of the Galaxy","url":"https://futurism.com/object-solar-system-center-galaxy","encrypted_content":"...","page_age":"1 day ago"},
# {"type":"web_search_result","title":"Astronomers Track Interstellar Comet 3I/ATLAS | Sci.News","url":"https://www.sci.news/astronomy/interstellar-comet-3i-atlas-14045.html","encrypted_content":"...","page_age":"2 days ago"},
# {"type":"web_search_result","title":"3I/ATLAS - Wikipedia","url":"https://en.wikipedia.org/wiki/3I/ATLAS","encrypted_content":"...","page_age":"12 hours ago"},
# {"type":"web_search_result","title":"NASA Discovers Interstellar Comet Moving Through Solar System - NASA Science","url":"https://science.nasa.gov/blogs/planetary-defense/2025/07/02/nasa-discovers-interstellar-comet-moving-through-solar-system/","encrypted_content":"...","page_age":"5 days ago"},
# {"type":"web_search_result","title":"Interstellar object 3I/ATLAS = C/2025 N1 (ATLAS): an image - 2 July 2025 - The Virtual Telescope Project 2.0","url":"https://www.virtualtelescope.eu/2025/07/03/interstellar-object-3i-atlas-c-2025-n1-atlas-an-image-2-july-2025/","encrypted_content":"...","page_age":"4 days ago"},
# {"type":"web_search_result","title":"A third interstellar object has been observed flying through our solar system | CNN","url":"https://www.cnn.com/2025/07/03/science/interstellar-object-comet-3i-atlas","encrypted_content":"...","page_age":"4 days ago"},
# {"type":"web_search_result","title":"The New Interstellar Object 3I/ATLAS is Smaller or Rarer Than it Looks | by Avi Loeb | Jul, 2025 | Medium","url":"https://avi-loeb.medium.com/the-new-interstellar-object-3i-atlas-is-smaller-or-rarer-than-it-looks-d2623ba1b1cf","encrypted_content":"...","page_age":"2 days ago"},
# {"type":"web_search_result","title":"Astronomers Discover Third-Ever Interstellar Object: 3I/ATLAS | Sci.News","url":"https://www.sci.news/astronomy/interstellar-object-3i-atlas-14041.html","encrypted_content":"...","page_age":"4 days ago"},
# {"type":"web_search_result","title":"NASA confirms that mysterious object shooting through the solar system is an 'interstellar visitor' — and it has a new name | Live Science","url":"https://www.livescience.com/space/comets/nasa-confirms-that-mysterious-object-shooting-through-the-solar-system-is-an-interstellar-visitor-and-it-has-a-new-name","encrypted_content":"...","page_age":"4 days ago"},
# {"type":"web_search_result","title":"New Interstellar Object Comet 3I/ATLAS—What We Know So Far as It Zips through the Solar System | Scientific American","url":"https://www.scientificamerican.com/article/new-interstellar-object-comet-3i-atlas-what-we-know-so-far-as-it-zips/","encrypted_content":"...","page_age":"4 days ago"}]},
# {"type":"text","text":"Now I'll search for the specific speed of 3I/ATLAS:"},
# {"type":"server_tool_use","id":"srvtoolu_01C9WvSpeBERiV6YJTsDu86G","name":"web_search","input":{"query":"3I/ATLAS speed velocity km/s"}},
# {"type":"web_search_tool_result","tool_use_id":"srvtoolu_01C9WvSpeBERiV6YJTsDu86G","content":[{"type":"web_search_result","title":"3I/ATLAS - Wikipedia","url":"https://en.wikipedia.org/wiki/3I/ATLAS","encrypted_content":"...","page_age":"12 hours ago"},
# {"type":"web_search_result","title":"Interstellar object 3I/ATLAS = C/2025 N1 (ATLAS): an image - 2 July 2025 - The Virtual Telescope Project 2.0","url":"https://www.virtualtelescope.eu/2025/07/03/interstellar-object-3i-atlas-c-2025-n1-atlas-an-image-2-july-2025/","encrypted_content":"...","page_age":"4 days ago"},
# {"type":"web_search_result","title":"It’s official! An interstellar object is visiting our solar system","url":"https://earthsky.org/space/new-interstellar-object-candidate-heading-toward-the-sun-a11pl3z/","encrypted_content":"...","page_age":"2 days ago"},
# {"type":"web_search_result","title":"NASA Discovers Interstellar Comet Moving Through Solar System - NASA Science","url":"https://science.nasa.gov/blogs/planetary-defense/2025/07/02/nasa-discovers-interstellar-comet-moving-through-solar-system/","encrypted_content":"...","page_age":"5 days ago"},
# {"type":"web_search_result","title":"Comet 3I/ATLAS - NASA Science","url":"https://science.nasa.gov/solar-system/comets/3i-atlas/","encrypted_content":"...","page_age":"4 days ago"},
# {"type":"web_search_result","title":"A third interstellar object has been observed flying through our solar system | CNN","url":"https://www.cnn.com/2025/07/03/science/interstellar-object-comet-3i-atlas","encrypted_content":"...","page_age":"4 days ago"},
# {"type":"web_search_result","title":"Interstellar Intruder: Why Comet 3I/ATLAS is a Game-Changer","url":"https://nasaspacenews.com/2025/07/interstellar-intruder-why-comet-3i-atlas-is-a-game-changer/","encrypted_content":"...","page_age":"1 day ago"},
# {"type":"web_search_result","title":"New Interstellar Object Comet 3I/ATLAS—What We Know So Far as It Zips through the Solar System | Scientific American","url":"https://www.scientificamerican.com/article/new-interstellar-object-comet-3i-atlas-what-we-know-so-far-as-it-zips/","encrypted_content":"...","page_age":"4 days ago"},
# {"type":"web_search_result","title":"Interstellar Comet 3I/ATLAS: Speed, Origin & Viewing Tips - memesita.com - Memesita","url":"https://www.memesita.com/interstellar-comet-3i-atlas-speed-origin-viewing-tips/","encrypted_content":"...","page_age":"4 days ago"},
# {"type":"web_search_result","title":"Newly spotted comet is third interstellar object seen in our solar system | Reuters","url":"https://www.reuters.com/science/newly-spotted-comet-is-third-interstellar-object-seen-our-solar-system-2025-07-03/","encrypted_content":"...","page_age":"4 days ago"}]},
# {"type":"text","text":"Based on my search results, I can now provide you with accurate information about the interstellar object you're asking about.\n\nYou mentioned \"3I/ATLAS\" which is indeed correct! This is a newly discovered interstellar comet that was just confirmed on July 1-2, 2025. Here's what we know about its speed:\n\n## Speed of Interstellar Comet 3I/ATLAS\n\n"},
# {"citations":[{"type":"web_search_result_location","cited_text":"Note that next April (2026) it will have a velocity relative to Earth of about 90 km/s. ","url":"https://earthsky.org/space/new-interstellar-object-candidate-heading-toward-the-sun-a11pl3z/","title":"It’s official! An interstellar object is visiting our solar system","encrypted_index":"EpABCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDOmLYGKMbWtyiLhlShoM86G3Xh1/bTkf0ZbyIjAZsOcxMxLNi/y9vOfvPRlaiz65CA5zstKMaHe2zt3vkTMSLDtRDw2i9FTIErCVHfMqFKqZtNTc+WMKNbupRSmj1pB+AadyGAQ="},
# {"type":"web_search_result_location","cited_text":"It poses no danger to Earth, coming no closer than 150 million miles (240 million km), which is more than 1.5 astronomical units (AU, or distance from...","url":"https://earthsky.org/space/new-interstellar-object-candidate-heading-toward-the-sun-a11pl3z/","title":"It’s official! An interstellar object is visiting our solar system","encrypted_index":"EpEBCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDKIQO7C/90kV/ILcyxoMyJhD8Uiv1zuPZm27IjBkQmNR73maqfsE24CFU0cvlpkmMCqz3eHQ3gM6Ye/yhQBq+2O2dqlt78CMZPkp0ykqFQZIvxI2kiupCQlPe1sxFCdxA/6aaRgE"}],
# "type":"text","text":"3I/ATLAS is currently moving at a relative speed of 61 km/s (38 mi/s)"},
# {"type":"text","text":" as it travels through our solar system. This translates to approximately:\n- "},
# {"citations":[{"type":"web_search_result_location","cited_text":"While Oumuamua and Borisov taught us profound lessons, 3I/ATLAS is larger (up to ~20 km), more active, and offers better observational clarity. ","url":"https://nasaspacenews.com/2025/07/interstellar-intruder-why-comet-3i-atlas-is-a-game-changer/","title":"Interstellar Intruder: Why Comet 3I/ATLAS is a Game-Changer","encrypted_index":"EpEBCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDOwwtANb7rN61+ZB6xoMfGChAZS08I758BfTIjAGRqW7wlE6rnCc0GixHpOOi234auQaACUxrvsihnvn8+VgSW1QEIM62NVfpk8SWZ8qFVTvh/IKPG+oh1YFr5mPkSiO/eeajxgE"}],
# "type":"text","text":"137,000 miles per hour (221,000 kilometers per hour)"},
# {"type":"text","text":"\n- "},
# {"citations":[{"type":"web_search_result_location","cited_text":"When it was discovered, the interstellar comet was traveling about 137,000 miles per hour (221,000 kilometers per hour, or 61 kilometers per second), ...","url":"https://science.nasa.gov/solar-system/comets/3i-atlas/","title":"Comet 3I/ATLAS - NASA Science","encrypted_index":"EpEBCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDKqJ70IZLHUsMCfRRhoMIYqh8pTAbbqr5o7GIjAomAqfrGwB3rhT9jb0zksKJOD6Ivb42q+kxOJseNWePaKbgOJahEQtqZReLLcABYUqFRiWy6Szo3zfKlrzY6zJVmyJ17NIthgE"}],
# "type":"text","text":"Roughly 134,000 mph (60 km/s) relative to the sun"},
# {"type":"text","text":"\n\nThe comet's speed will vary as it approaches the Sun. "},
# {"citations":[{"type":"web_search_result_location","cited_text":"Arriving from the direction of the constellation Sagittarius, the interstellar comet has been officially named 3I/ATLAS. ","url":"https://science.nasa.gov/blogs/planetary-defense/2025/07/02/nasa-discovers-interstellar-comet-moving-through-solar-system/","title":"NASA Discovers Interstellar Comet Moving Through Solar System - NASA Science","encrypted_index":"EpABCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDPOQ/CCsPw/NAw+1hRoMeSTG7pUvLv9jePkhIjDxfQuKdvdyB2EiwMxnXmQ1nbVSDGEzekwuRYdQIq9MxXQ0CFEKyEltMAVmRA5bMi4qFKvEpPkVBqpj2S/f22RFCJEb912xGAQ="},
# {"type":"web_search_result_location","cited_text":"On July 1, the NASA-funded ATLAS (Asteroid Terrestrial-impact Last Alert System) survey telescope in Rio Hurtado, Chile, first reported observations o...","url":"https://science.nasa.gov/blogs/planetary-defense/2025/07/02/nasa-discovers-interstellar-comet-moving-through-solar-system/","title":"NASA Discovers Interstellar Comet Moving Through Solar System - NASA Science","encrypted_index":"EpABCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDE7PyFNo1+fBLfSQ8xoMPRkVSdev0apr9705IjCVCWGXcwN5m32vgPvTZYaHF/OF6YM0lOUGf2R1bki1WAwELLU8HxqZbbfS6beiMQoqFCF85j9lsqvhrlDwTzqxIeZb0cuMGAQ="}],
# "type":"text","text":"At perihelion (closest point to the Sun) on October 29, it will reach a speed of about 68 km/s"},
# {"type":"text","text":".\n\n## Key Facts About 3I/ATLAS\n\n- **Discovery**: "},
# {"citations":[{"type":"web_search_result_location","cited_text":"Its discovery is sparking excitement and scientific urgency, marking a milestone in cosmic exploration. ","url":"https://nasaspacenews.com/2025/07/interstellar-intruder-why-comet-3i-atlas-is-a-game-changer/","title":"Interstellar Intruder: Why Comet 3I/ATLAS is a Game-Changer","encrypted_index":"EpABCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDAj2td4hdlN6iSyyVRoMB/dJSsBYlHMO3B5ZIjA7npjUt0H9twPXHVd3F6g9NBqDUX+O19DqqNdnaNvw5xZgXJDC/tRNRFVmH0L4WQUqFLVmyDxtgbLc2iCWZj4Cqu+noQL/GAQ="}],
# "type":"text","text":"First reported by the NASA-funded ATLAS survey telescope in Rio Hurtado, Chile, on July 1, 2025"},
# {"type":"text","text":"\n- **Current Distance**: "},
# {"citations":[{"type":"web_search_result_location","cited_text":"It’s moving at roughly 60 km/s (over 200,000 km/h) in a hyperbolic escape path. ","url":"https://nasaspacenews.com/2025/07/interstellar-intruder-why-comet-3i-atlas-is-a-game-changer/","title":"Interstellar Intruder: Why Comet 3I/ATLAS is a Game-Changer","encrypted_index":"EpABCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDF6F3p5i5eLu+mpY3xoMhxz66K5ZPkU1dcvqIjCUokHGKHRjbmyz/I/Sz6vw725in988mgI/KGvzBuJOXhB8j9JyBPRKEPmVBB5+ulMqFN50ixYfzc7W50q93ecSlIxb09CzGAQ="}],
# "type":"text","text":"About 410 million miles (670 million kilometers) away from the Sun"},
# {"type":"text","text":"\n- **Official Designation**: "},
# {"citations":[{"type":"web_search_result_location","cited_text":"","url":"https://science.nasa.gov/blogs/planetary-defense/2025/07/02/nasa-discovers-interstellar-comet-moving-through-solar-system/","title":"NASA Discovers Interstellar Comet Moving Through Solar System - NASA Science","encrypted_index":"EpABCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDBfnGA9O9L/xKLW7QBoMzXlik+STgPA/SJ82IjClROgEVes/1KzPK9wm5MWgBcUnlHgd4WxIxoSHnWahGCEJiulvd7VAgIMyEyFDJAkqFN+nAspvc42oO5U1v51N2bC5dYsLGAQ="}],
# "type":"text","text":"C/2025 N1 (ATLAS)"},
# {"type":"text","text":"\n- **Significance**: "},
# {"citations":[{"type":"web_search_result_location","cited_text":"","url":"https://science.nasa.gov/blogs/planetary-defense/2025/07/02/nasa-discovers-interstellar-comet-moving-through-solar-system/","title":"NASA Discovers Interstellar Comet Moving Through Solar System - NASA Science","encrypted_index":"EpEBCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDDa2cqBZVAVXmOkYQBoMHF8ooLI0r+Mu2Xj7IjBt2HZrWT7SZQk6kwna2hdiwY/zgvURx2dfkwJjya5cAGVFgCxGvY6bxKU1fP0pPQsqFVNYDlnfTV/EN/IZMjsZo7UyCb+j3BgE"}],
# "type":"text","text":"It is the third interstellar object confirmed passing through the Solar System, after 1I/ʻOumuamua (discovered 19 October 2017) and 2I/Borisov (discovered 29 August 2019)"},
# {"type":"text","text":"\n\n## Comparison with Other Interstellar Objects\n\nFor context, the previous interstellar comet 2I/Borisov had:\n- "},
# {"citations":[{"type":"web_search_result_location","cited_text":"The high hyperbolic excess velocity of 2I/Borisov of 32 km/s makes it hard for a spacecraft to reach the comet with existing technology: according to ...","url":"https://en.wikipedia.org/wiki/2I/Borisov","title":"2I/Borisov - Wikipedia","encrypted_index":"Eo8BCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDH4RNegKf14SPGwFrRoMIlGrOBwauOYYRpj8IjAW9+Oz05N6vf5JQUqQms3sNRTct9sTMOsOLqEvq2tjezbSw/cZWcyrvTEnef+plKwqE1jaIvLS6eJal82BmT3+ja490XcYBA=="}],
# "type":"text","text":"A hyperbolic excess velocity of 32 km/s"},
# {"type":"text","text":"\n- "},
# {"citations":[{"type":"web_search_result_location","cited_text":"The Sun&#x27;s gravity is slightly changing comet 2I/Borisov’s trajectory, but it cannot capture it and pull it into orbit because of the comet’s path...","url":"https://science.nasa.gov/solar-system/comets/2i-borisov/","title":"Comet 2I/Borisov - NASA Science","encrypted_index":"Eo8BCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDJxxGywSgdxz+ooesRoM0Wfq5aWcWzhueTlFIjAzgO6+lEFVtdGeAy5rCAfdRsxIEF4J9qBekqDm8ICnkz0IUHlaxJ60HuD2H3PRoyIqE2UewJ5cI27VChJvlkBY6/Chk3MYBA=="},
# {"type":"web_search_result_location","cited_text":"The Sun's gravity is slightly changing comet 2I/Borisov’s trajectory, but it cannot capture it and pull it into orbit because of the comet’s path and ...","url":"https://science.nasa.gov/solar-system/comets/2i-borisov/","title":"Comet 2I/Borisov - NASA Science","encrypted_index":"Eo8BCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDGpZeWjHWJmKw+WwoRoMKZ1+CmQVkGjqhny0IjBidW6YoEOLw1jOQ8s6/poDSd3OMJBGAXx1PIxGYmgafoX1UtAMrwX+sYlKMOjcT/MqE6LqZRRW98iv+X4ddq8LM+Sq9TwYBA=="}],
# "type":"text","text":"A velocity of about 110,000 miles (177,000 kilometers) per hour"},
# {"type":"text","text":"\n\nThis makes "},
# {"citations":[{"type":"web_search_result_location","cited_text":"If you&#x27;re enjoying this article, consider supporting our award-winning journalism by subscribing. ","url":"https://www.scientificamerican.com/article/new-interstellar-object-comet-3i-atlas-what-we-know-so-far-as-it-zips/","title":"New Interstellar Object Comet 3I/ATLAS—What We Know So Far as It Zips through the Solar System | Scientific American","encrypted_index":"EpABCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDIiiaAns5a4dkNhWVxoM8AAqO4oAGXW1KDqmIjBu0r2/ZZ9KmT+Jc0jCfK2f7txf8qXNziPn5UBsif/Axn/rNxrgPMKNmOzDodpRoU4qFA4F5aHwYemWnIH7j894Y6/RYtaBGAQ="},
# {"type":"web_search_result_location","cited_text":"The object will make its closest approach to the sun around October 30, reaching a distance of about 210 million kilometers, just within the orbit of ...","url":"https://www.scientificamerican.com/article/new-interstellar-object-comet-3i-atlas-what-we-know-so-far-as-it-zips/","title":"New Interstellar Object Comet 3I/ATLAS—What We Know So Far as It Zips through the Solar System | Scientific American","encrypted_index":"EpEBCioIBRgCIiRkYjc2NTdjMy05YWVjLTQxMTYtOTZiNi1mNWI3ZWNhNTdiYWYSDDzWuA0pUv8H2Na/CxoMk+kw1oJzcDCWIRqMIjDg4cDy/SuavH6UP7qhAdMgeiZJI/vfgkcZtEQxAZJEVDrLgTuWQxQkvZuLFXv7J1cqFb3y3dGLFY+llQAM2lhDpG5otpMbGxgE"}],
# "type":"text","text":"3I/ATLAS the brightest and fastest of the three interstellar objects discovered so far"},
# {"type":"text","text":"."}],"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":81872,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":985,"service_tier":"standard","server_tool_use":{"web_search_requests":3}}}
