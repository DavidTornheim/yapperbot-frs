# yapperbot-frs
Bot that powers the [Feedback Request Service](https://en.wikipedia.org/wiki/WP:FRS) on Wikipedia
(This is a copy to experiment with)

# NOTES on configuration files:
ybtools needs either of these files:
  config.yml
  config-global.yml
to load this data type:
    type configObject struct {
    	APIEndpoint string
    	BotUsername string
    }

yapperbot-frs needs:
  config-FRS.yml


      
