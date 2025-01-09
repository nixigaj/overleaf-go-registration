const puppeteer = require('puppeteer');

async function registerUser(overleafUrl, adminEmail, adminPassword, userEmail) {
    const browser = await puppeteer.launch({
        headless: "new",
        args: ['--no-sandbox']
    });

    const page = await browser.newPage();

    try {
        // Check if overleaf.sid cookie exists and is valid
        const cookies = await page.cookies();
        const sidCookie = cookies.find(cookie => cookie.name === 'overleaf.sid');

        if (sidCookie && new Date(sidCookie.expires * 1000) > new Date()) {
            // Cookie exists and has not expired, navigate to admin register page
            await page.goto(`${overleafUrl}/admin/register`);
        } else {
            // Login as admin
            await page.goto(`${overleafUrl}/login`);
            await page.waitForSelector('input[type="email"]');

            // Fill in login form
            await page.type('input[type="email"]', adminEmail);
            await page.type('input[type="password"]', adminPassword);
            await page.click('button[type="submit"]');

            // Wait for login to complete
            await page.waitForNavigation();

            // Go to admin registration page
            await page.goto(`${overleafUrl}/admin/register`);
        }

        // Wait for the registration form
        await page.waitForSelector('#user-activate-register-container .card form input[name="email"]');

        // Fill in and submit registration form
        await page.type('#user-activate-register-container .card form input[name="email"]', userEmail);
        await page.click('#user-activate-register-container .card form button.btn-primary');

        // Wait for success message or confirmation table
        await page.waitForSelector('#user-activate-register-container .card .row-spaced.text-success', { timeout: 5000 });

        const successMessage = await page.$eval('#user-activate-register-container .card .row-spaced.text-success', el => el.textContent.trim());
        console.log(JSON.stringify({ success: true, message: successMessage }));

        await browser.close();
        process.exit(0);

    } catch (error) {
        console.error(JSON.stringify({
            success: false,
            error: error.message,
            overleafUrl: overleafUrl,
            pageContent: await page.content()
        }));
        await browser.close();
        process.exit(1);
    }
}

// Get arguments from command line
const [,, overleafUrl, adminEmail, adminPassword, userEmail] = process.argv;

if (!overleafUrl || !adminEmail || !adminPassword || !userEmail) {
    console.error(JSON.stringify({
        success: false,
        error: "Missing required arguments"
    }));
    process.exit(1);
}

registerUser(overleafUrl, adminEmail, adminPassword, userEmail);
