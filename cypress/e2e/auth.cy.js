describe('auth spec', () => {
  it('can login with a preconfigured account', () => {
    cy.visit('/accounts/login')
    cy.get('#username').type('joe')
    cy.get('#password').type('joe')
    cy.get('.btn-primary').click()
    cy.contains('footer', 'User: joe').should('be.visible')
    cy.clearCookies()
  })
  it('can logout after logging in', () => {
    cy.visit('/accounts/login')
    cy.get('#username').type('joe')
    cy.get('#password').type('joe')
    cy.get('.btn-primary').click()
    cy.contains('footer', 'User: joe').should('be.visible')

    cy.contains('header a', 'Logout').should('be.visible')
    cy.get('.logout-link').click()

    cy.contains('footer', 'User: joe').should('not.exist')

    cy.clearCookies()
  })
  it('creates account and can login', () => {
    Cypress.Cookies.debug(true)
    cy.visit('/accounts/new')
    cy.contains('Create your account').should('be.visible')
    const random = Math.random().toString().substr(2, 9)
    const username = 'joe-'+random
    cy.get('#username').type(username)
    cy.get('#password').type('pass')
    cy.get('#password2').type('pass')
    cy.get('.btn-primary').click()
    cy.contains('Account created').should('be.visible')
    
    cy.clearCookies() // for some reason cypress does not update the cookie in the next page

    cy.visit('/accounts/login')
    cy.get('input[name=_csrf]').should('not.be.visible')
    cy.get('#username').type(username)
    cy.get('#password').type('pass')
    cy.get('.btn-primary').click()
    cy.contains('footer', `User: ${username}`).should('be.visible')
  })
  it('fails to create account because username is empty', () => {
    cy.visit('/accounts/new')
    cy.contains('Create your account').should('be.visible')
    const random = Math.random().toString().substr(2, 9)
    cy.get('#password').type('pass')
    cy.get('#password2').type('pass')
    cy.get('.btn-primary').click()
    cy.contains('username is empty').should('be.visible')
  })
  it('fails to create account because passwords do not match', () => {
    cy.visit('/accounts/new')
    cy.contains('Create your account').should('be.visible')
    const random = Math.random().toString().substr(2, 9)
    cy.get('#username').type('joe-'+random)
    cy.get('#password').type('this')
    cy.get('#password2').type('that')
    cy.get('.btn-primary').click()
    cy.contains('passwords do not match').should('be.visible')
  })
  it('fails to create account because username exists', () => {
    cy.visit('/accounts/new')
    cy.contains('Create your account').should('be.visible')
    cy.get('#username').type('joe')
    cy.get('#password').type('joe')
    cy.get('#password2').type('joe')
    cy.get('.btn-primary').click()
    cy.contains('username is not available').should('be.visible')
  })
})