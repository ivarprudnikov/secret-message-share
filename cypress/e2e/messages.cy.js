describe('messages spec', () => {
  it('joe can see his message in the list', () => {
    cy.loginJoe()
    cy.visit('/messages')
    cy.contains('h1', 'Messages').should('be.visible')
    cy.get('.message-row').should('have.length', 1)
  })
  it('joe can share his message with anonymous user', () => {
    cy.loginJoe()
    
    cy.visit('/messages')
    cy.get('.message-row a')
      .should('have.attr', 'href')
      .then(($href) => {
        cy.logout()
        cy.visit($href)

        cy.contains('h1', 'Secret message').should('be.visible')
      })
  })

  it('alice creates a message, shares it and annon can decrypt it, then message is deleted', function() {
    
    cy.loginAlice()

    // create a message
    cy.get('.nav-link.messages-new').click()
    cy.contains('h3', 'Create new').should('be.visible')
    cy.get('#payload').type('foobar')
    cy.get('.btn-primary').click()
    cy.contains('h5', 'Message securely stored').should('be.visible')
    cy.get('.message-pin')
      .invoke('text')
      .then(parseFloat)
      .as('pinval')
    cy.get('@pinval').should('be.gte', 100)
    cy.get('.message-link').should('have.attr', 'href')
      .as('messageHref')
      .then(($href) => {
        cy.logout()
      })

    cy.get('@messageHref').then($href => {
      cy.get('@pinval').then((pinval) => {
        cy.enterPin($href, pinval)
      })
    })

    cy.contains('.message-content-decrypted', 'foobar').should('be.visible')
    cy.get('input#pin').should('not.exist')
    
    cy.get('@messageHref')
      .then(($href) => {
        cy.visit($href, {failOnStatusCode: false})
        cy.contains('h1', '404: Page not found').should('be.visible')
      })
  })

  it('new message gets deleted after unuccessful PIN attempts', function() {
    
    Cypress.Cookies.debug(true)

    cy.loginAlice()

    // create a message
    cy.get('.nav-link.messages-new').click()
    cy.contains('h3', 'Create new').should('be.visible')
    cy.get('#payload').type('foobar')
    cy.get('.btn-primary').click()
    cy.contains('h5', 'Message securely stored').should('be.visible')
    cy.get('.message-pin')
      .invoke('text')
      .then(parseFloat)
      .as('pinval')
    cy.get('@pinval').should('be.gte', 100)
    cy.get('.message-link').should('have.attr', 'href')
      .as('messageHref')
      .then(($href) => {
        cy.logout()
      })
    
    // pin attempt
    cy.get('@messageHref').then($href => {
      cy.enterPin($href, '0000')
    })
    cy.contains('p', 'failed to get a message').should('be.visible')
    // pin attempt
    cy.get('@messageHref').then($href => {
      cy.enterPin($href, '0000')
    })
    cy.contains('p', 'failed to get a message').should('be.visible')
    // pin attempt
    cy.get('@messageHref').then($href => {
      cy.enterPin($href, '0000')
    })
    cy.contains('p', 'failed to get a message').should('be.visible')
    // pin attempt
    cy.get('@messageHref').then($href => {
      cy.enterPin($href, '0000')
    })
    cy.contains('p', 'failed to get a message').should('be.visible')
    // pin attempt
    cy.get('@messageHref').then($href => {
      cy.enterPin($href, '0000')
    })
    cy.contains('p', 'failed to get a message').should('be.visible')
    // message should have been deleted
    cy.get('@messageHref').then($href => {
      cy.visit($href, {failOnStatusCode: false})
      cy.contains('h1', '404: Page not found').should('be.visible')
    })

  })

})